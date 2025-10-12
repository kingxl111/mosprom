package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/kingxl111/mosprom/ApiGateway/internal/config"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
)

type Processor interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type processor struct {
	config       config.KafkaConfig
	redisClient  redis.Client
	logger       *slog.Logger
	consumer     sarama.ConsumerGroup
	workers      []*worker
	workerCount  int
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
}

type worker struct {
	id        string
	processor *processor
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
}

type FileProcessingEvent struct {
	FileID    string `json:"file_id"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func NewProcessor(cfg config.KafkaConfig, redisClient redis.Client, logger *slog.Logger) Processor {
	return &processor{
		config:       cfg,
		redisClient:  redisClient,
		logger:       logger,
		workerCount:  cfg.Workers(),
		shutdownChan: make(chan struct{}),
	}
}

func (p *processor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("processor is already running")
	}

	// Создаем Kafka consumer group
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true
	config.Version = sarama.V2_8_0_0

	consumer, err := sarama.NewConsumerGroup(p.config.Brokers(), p.config.GroupID(), config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	p.consumer = consumer

	// Создаем worker'ы
	p.workers = make([]*worker, p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		workerCtx, cancel := context.WithCancel(ctx)
		w := &worker{
			id:        fmt.Sprintf("worker-%d", i),
			processor: p,
			ctx:       workerCtx,
			cancel:    cancel,
			done:      make(chan struct{}),
		}
		p.workers[i] = w
	}

	p.running = true

	// Запускаем consumer group
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.runConsumerGroup(ctx)
	}()

	// Запускаем worker'ы
	for _, w := range p.workers {
		p.wg.Add(1)
		go func(worker *worker) {
			defer p.wg.Done()
			worker.run()
		}(w)
	}

	p.logger.Info("event processor started",
		slog.Int("workers", p.workerCount),
		slog.String("group_id", p.config.GroupID()),
		slog.String("topic", p.config.Topic()))

	return nil
}

func (p *processor) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.logger.Info("stopping event processor...")

	// Сигнализируем о завершении
	close(p.shutdownChan)

	// Останавливаем worker'ы
	for _, w := range p.workers {
		w.cancel()
	}

	// Останавливаем consumer group
	if p.consumer != nil {
		if err := p.consumer.Close(); err != nil {
			p.logger.Error("failed to close consumer group", slog.Any("err", err))
		}
	}

	// Ждем завершения всех горутин
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("all workers stopped gracefully")
	case <-ctx.Done():
		p.logger.Warn("timeout waiting for workers to stop")
	}

	p.running = false
	return nil
}

func (p *processor) runConsumerGroup(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("consumer group context cancelled")
			return
		case <-p.shutdownChan:
			p.logger.Info("consumer group shutdown requested")
			return
		default:
			// Запускаем consumer group
			handler := &consumerGroupHandler{processor: p}
			err := p.consumer.Consume(ctx, []string{p.config.Topic()}, handler)
			if err != nil {
				p.logger.Error("error consuming messages", slog.Any("err", err))
				time.Sleep(time.Second)
			}
		}
	}
}

func (w *worker) run() {
	w.processor.logger.Info("worker started", slog.String("worker_id", w.id))

	for {
		select {
		case <-w.ctx.Done():
			w.processor.logger.Info("worker stopping", slog.String("worker_id", w.id))
			close(w.done)
			return
		case <-w.processor.shutdownChan:
			w.processor.logger.Info("worker shutdown requested", slog.String("worker_id", w.id))
			close(w.done)
			return
		default:
			// Worker ждет сообщения от consumer group
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// consumerGroupHandler реализует sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	processor *processor
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.processor.logger.Info("consumer group session setup")
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.processor.logger.Info("consumer group session cleanup")
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			h.processor.logger.Info("received message",
				slog.String("topic", message.Topic),
				slog.Int("partition", int(message.Partition)),
				slog.Int64("offset", message.Offset),
				slog.String("worker_id", "consumer"))

			// Обрабатываем сообщение
			if err := h.processMessage(session.Context(), message); err != nil {
				h.processor.logger.Error("failed to process message",
					slog.Any("err", err),
					slog.String("topic", message.Topic),
					slog.Int64("offset", message.Offset))
			}

			// Отмечаем сообщение как обработанное
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *consumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	// Парсим сообщение
	var event FileProcessingEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	h.processor.logger.Info("processing file event",
		slog.String("file_id", event.FileID),
		slog.String("filename", event.Filename),
		slog.String("status", event.Status))

	// Имитируем обработку файла
	time.Sleep(2 * time.Second)

	// Обновляем статус в Redis
	event.Status = "processed"
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal updated event: %w", err)
	}

	// Публикуем уведомление о готовности файла
	if err := h.processor.redisClient.Publish(ctx, "file_processed", string(eventJSON)); err != nil {
		return fmt.Errorf("failed to publish file processed event: %w", err)
	}

	h.processor.logger.Info("file processing completed",
		slog.String("file_id", event.FileID))

	return nil
}
