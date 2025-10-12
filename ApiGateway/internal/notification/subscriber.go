package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
)

type Subscriber interface {
	Start(ctx context.Context) error
	Stop() error
	SubscribeToFileEvents(ctx context.Context, callback FileEventCallback) error
}

type FileEventCallback func(event FileEvent) error

type FileEvent struct {
	FileID    string `json:"file_id"`
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Message   string `json:"message"`
	ResultURL string `json:"result_url,omitempty"`
	Timestamp string `json:"timestamp"`
}

type subscriber struct {
	redisClient redis.Client
	logger      *slog.Logger
	pubsub      redis.PubSub
	callbacks   map[string]FileEventCallback
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

func NewSubscriber(redisClient redis.Client, logger *slog.Logger) Subscriber {
	return &subscriber{
		redisClient: redisClient,
		logger:      logger,
		callbacks:   make(map[string]FileEventCallback),
		stopChan:    make(chan struct{}),
	}
}

func (s *subscriber) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("subscriber is already running")
	}

	s.running = true

	// Подписываемся на каналы уведомлений
	channels := []string{
		"file_uploaded",
		"file_processing",
		"file_processed",
		"file_failed",
	}

	s.pubsub = s.redisClient.Subscribe(ctx, channels...)

	// Запускаем обработчик сообщений
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleMessages(ctx)
	}()

	s.logger.Info("notification subscriber started",
		slog.Any("channels", channels))

	return nil
}

func (s *subscriber) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("stopping notification subscriber...")

	// Сигнализируем о завершении
	close(s.stopChan)

	// Закрываем pubsub
	if s.pubsub != nil {
		if err := s.pubsub.Close(); err != nil {
			s.logger.Error("failed to close pubsub", slog.Any("err", err))
		}
	}

	// Ждем завершения горутин
	s.wg.Wait()

	s.running = false
	s.logger.Info("notification subscriber stopped")
	return nil
}

func (s *subscriber) SubscribeToFileEvents(ctx context.Context, callback FileEventCallback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Генерируем уникальный ID для callback'а
	callbackID := fmt.Sprintf("callback_%d", time.Now().UnixNano())
	s.callbacks[callbackID] = callback

	s.logger.Info("file event callback registered", slog.String("callback_id", callbackID))
	return nil
}

func (s *subscriber) handleMessages(ctx context.Context) {
	ch := s.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("subscriber context cancelled")
			return
		case <-s.stopChan:
			s.logger.Info("subscriber shutdown requested")
			return
		case msg, ok := <-ch:
			if !ok {
				s.logger.Info("pubsub channel closed")
				return
			}

			s.logger.Debug("received notification",
				slog.String("channel", msg.Channel),
				slog.String("payload", msg.Payload))

			// Парсим событие
			var event FileEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				s.logger.Error("failed to unmarshal notification",
					slog.Any("err", err),
					slog.String("payload", msg.Payload))
				continue
			}

			// Устанавливаем timestamp если не установлен
			if event.Timestamp == "" {
				event.Timestamp = time.Now().Format(time.RFC3339)
			}

			// Вызываем все зарегистрированные callback'и
			s.mu.RLock()
			for callbackID, callback := range s.callbacks {
				go func(id string, cb FileEventCallback, evt FileEvent) {
					if err := cb(evt); err != nil {
						s.logger.Error("callback execution failed",
							slog.String("callback_id", id),
							slog.Any("err", err))
					}
				}(callbackID, callback, event)
			}
			s.mu.RUnlock()
		}
	}
}

// PublishFileEvent публикует событие о файле
func (s *subscriber) PublishFileEvent(ctx context.Context, channel string, event FileEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := s.redisClient.Publish(ctx, channel, string(eventJSON)); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	s.logger.Info("file event published",
		slog.String("channel", channel),
		slog.String("file_id", event.FileID),
		slog.String("status", event.Status))

	return nil
}

// GetActiveSubscriptions возвращает информацию об активных подписках
func (s *subscriber) GetActiveSubscriptions() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscriptions := make(map[string]interface{})
	subscriptions["callback_count"] = len(s.callbacks)
	subscriptions["running"] = s.running

	return subscriptions
}
