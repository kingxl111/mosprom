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

	"github.com/kingxl111/mosprom/AuthService/internal/config"
	env "github.com/kingxl111/mosprom/AuthService/internal/environment"
	httpserver "github.com/kingxl111/mosprom/AuthService/internal/gates/http-server"
	"github.com/kingxl111/mosprom/AuthService/internal/repository/postgres"
	"github.com/kingxl111/mosprom/AuthService/internal/user/service"
	authapi "github.com/kingxl111/mosprom/AuthService/pkg/api/auth"
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

	// PostgreSQL config
	pgConfig, err := config.NewPGConfig()
	if err != nil {
		return fmt.Errorf("pg config: %w", err)
	}

	db, err := postgres.NewDB(
		pgConfig.Username,
		pgConfig.Password,
		pgConfig.Host,
		pgConfig.Port,
		pgConfig.DBName,
		pgConfig.SSLMode,
	)
	if err != nil {
		return fmt.Errorf("db init: %w", err)
	}
	defer db.Close()

	// Настройка логгера
	loggerConfig, err := config.NewLoggerConfig()
	if err != nil {
		return fmt.Errorf("logger config: %w", err)
	}

	handlerOpts := &slog.HandlerOptions{Level: loggerConfig.Level()}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOpts))

	// Репозиторий
	repo := postgres.NewRepository(db)

	// Сервис авторизации
	authService := service.NewAuthService(repo)

	// HTTP config
	httpConfig, err := config.NewHTTPConfig()
	if err != nil {
		return fmt.Errorf("http config: %w", err)
	}

	// HTTP обработчики (API)
	handler := httpserver.NewHandler(authService, logger)
	mux := http.NewServeMux()
	apiHandler := authapi.HandlerFromMux(handler, mux)

	// Настраиваем сервер и middleware
	var opts env.ServerOptions
	opts.WithLogger(logger)
	httpServer := opts.NewServer(apiHandler, httpConfig.Address())

	// graceful shutdown
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		logger.Info("starting HTTP server...", slog.String("addr", httpConfig.Address()))
		if err := env.ListenAndServeContext(ctx, httpServer); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
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
