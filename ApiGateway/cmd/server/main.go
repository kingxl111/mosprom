package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kingxl111/mosprom/ApiGateway/internal/config"
	env "github.com/kingxl111/mosprom/ApiGateway/internal/environment"
	"github.com/kingxl111/mosprom/ApiGateway/internal/event"
	httpserver "github.com/kingxl111/mosprom/ApiGateway/internal/gates/http-server"
	"github.com/kingxl111/mosprom/ApiGateway/internal/monitoring"
	"github.com/kingxl111/mosprom/ApiGateway/internal/notification"
	"github.com/kingxl111/mosprom/ApiGateway/internal/proxy"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	defaultLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(defaultLogger)

	if err := runMain(ctx); err != nil {
		defaultLogger.Error("run main", slog.Any("err", err))
		os.Exit(1)
	}
}

func runMain(ctx context.Context) error {
	flag.Parse()

	// Загружаем .env
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Настройка логгера
	loggerConfig, err := config.NewLoggerConfig()
	if err != nil {
		return fmt.Errorf("logger config: %w", err)
	}

	handlerOpts := &slog.HandlerOptions{Level: loggerConfig.Level()}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOpts))

	// Redis клиент
	redisConfig, err := config.NewRedisConfig()
	if err != nil {
		return fmt.Errorf("redis config: %w", err)
	}

	redisClient, err := redis.NewClient(redisConfig)
	if err != nil {
		return fmt.Errorf("redis client: %w", err)
	}
	defer redisClient.Close()

	// Kafka конфигурация
	kafkaConfig, err := config.NewKafkaConfig()
	if err != nil {
		return fmt.Errorf("kafka config: %w", err)
	}

	// HTTP config
	httpConfig, err := config.NewHTTPConfig()
	if err != nil {
		return fmt.Errorf("http config: %w", err)
	}

	// AuthService proxy
	authServiceConfig, err := config.NewAuthServiceConfig()
	if err != nil {
		return fmt.Errorf("auth service config: %w", err)
	}

	authProxy := proxy.NewAuthServiceProxy(authServiceConfig, logger)

	// HTTP обработчики (API Gateway)
	handler := httpserver.NewHandler(authProxy, redisClient, logger)
	mux := http.NewServeMux()

	// Проксируем все запросы к AuthService
	mux.Handle("/api/v1/auth/", http.StripPrefix("/api/v1/auth", authProxy.ProxyHandler()))

	// Обработка файлов
	mux.HandleFunc("/api/v1/files/upload", handler.UploadFile)
	mux.HandleFunc("/api/v1/files/status/", handler.GetFileStatus)

	// Настраиваем сервер и middleware
	var opts env.ServerOptions
	opts.WithLogger(logger)
	httpServer := opts.NewServer(mux, httpConfig.Address())

	// Система обработки событий
	eventProcessor := event.NewProcessor(kafkaConfig, redisClient, logger)

	// Notification subscriber
	notificationSubscriber := notification.NewSubscriber(redisClient, logger)

	// Мониторинг горутин
	goroutineMonitor := monitoring.NewGoroutineMonitor(logger)

	// graceful shutdown
	eg, ctx := errgroup.WithContext(ctx)

	// HTTP сервер
	eg.Go(func() error {
		logger.Info("starting HTTP server...", slog.String("addr", httpConfig.Address()))
		if err := env.ListenAndServeContext(ctx, httpServer); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	// Event processor
	eg.Go(func() error {
		logger.Info("starting event processor...")
		return eventProcessor.Start(ctx)
	})

	// Notification subscriber
	eg.Go(func() error {
		logger.Info("starting notification subscriber...")
		return notificationSubscriber.Start(ctx)
	})

	// Goroutine monitor
	eg.Go(func() error {
		logger.Info("starting goroutine monitor...")
		return goroutineMonitor.Start(ctx)
	})

	// Graceful shutdown
	eg.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down server...")

		// Останавливаем event processor
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := eventProcessor.Stop(shutdownCtx); err != nil {
			logger.Error("error during event processor shutdown", slog.Any("err", err))
		}

		// Останавливаем notification subscriber
		if err := notificationSubscriber.Stop(); err != nil {
			logger.Error("error during notification subscriber shutdown", slog.Any("err", err))
		}

		// Останавливаем HTTP сервер
		httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
			logger.Error("error during server shutdown", slog.Any("err", err))
			return err
		}

		logger.Info("server stopped gracefully")
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Error("server terminated with error", slog.Any("err", err))
		return fmt.Errorf("server terminated: %w", err)
	}

	logger.Info("server exited cleanly ✅")
	return nil
}
