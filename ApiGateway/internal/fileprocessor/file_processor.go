package fileprocessor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/kingxl111/mosprom/ApiGateway/internal/config"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
)

var _ FileProcessor = (*fileProcessor)(nil)

type FileProcessor interface {
	ProcessUpload(ctx context.Context, file io.Reader, filename string, userID string) (*FileMetadata, error)
	//GetFileStatus(ctx context.Context, fileID string) (*FileStatus, error)
	//ListFiles(ctx context.Context, userID string, page, limit int) (*FileList, error)
	//DeleteFile(ctx context.Context, fileID, userID string) error
}

type FileMetadata struct {
	FileID    string    `json:"file_id"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Hash      string    `json:"hash"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	DataType  string    `json:"data_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FileStatus struct {
	FileID       string    `json:"file_id"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	Message      string    `json:"message"`
	ErrorDetails string    `json:"error_details,omitempty"`
	ResultURL    string    `json:"result_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProcessedAt  time.Time `json:"processed_at,omitempty"`
}

type FileList struct {
	Files      []FileItem `json:"files"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

type FileItem struct {
	FileID      string    `json:"file_id"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	Status      string    `json:"status"`
	DataType    string    `json:"data_type"`
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
}

type fileProcessor struct {
	kafkaConfig config.KafkaConfig
	redisClient redis.Client
	logger      *slog.Logger
	producer    sarama.SyncProducer
}

func NewFileProcessor(kafkaConfig config.KafkaConfig, redisClient redis.Client, logger *slog.Logger) (FileProcessor, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionLZ4
	config.Producer.Flush.Bytes = kafkaConfig.BatchSize()
	config.Producer.Flush.Frequency = time.Duration(kafkaConfig.LingerMs()) * time.Millisecond

	producer, err := sarama.NewSyncProducer(kafkaConfig.Brokers(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &fileProcessor{
		kafkaConfig: kafkaConfig,
		redisClient: redisClient,
		logger:      logger,
		producer:    producer,
	}, nil
}

func (fp *fileProcessor) ProcessUpload(ctx context.Context, file io.Reader, filename string, userID string) (*FileMetadata, error) {
	fileID := uuid.New().String()

	// Читаем файл и вычисляем хеш для идемпотентности
	hasher := sha256.New()
	teeReader := io.TeeReader(file, hasher)

	// Здесь должна быть логика сохранения файла в S3/MinIO
	// Пока используем временное решение - просто читаем для вычисления хеша
	content, err := io.ReadAll(teeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileHash := hex.EncodeToString(hasher.Sum(nil))

	metadata := &FileMetadata{
		FileID:    fileID,
		Filename:  filename,
		Size:      int64(len(content)),
		Hash:      fileHash,
		UserID:    userID,
		Status:    "uploaded",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Сохраняем метаданные в Redis
	if err := fp.saveMetadata(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Отправляем событие в Kafka для ETL обработки
	if err := fp.sendToKafka(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to send to kafka: %w", err)
	}

	return metadata, nil
}

func (fp *fileProcessor) saveMetadata(ctx context.Context, metadata *FileMetadata) error {
	// Сохраняем метаданные в Redis с TTL 24 часа
	key := fmt.Sprintf("file:%s", metadata.FileID)
	// ... реализация сохранения в Redis
	return nil
}

func (fp *fileProcessor) sendToKafka(ctx context.Context, metadata *FileMetadata) error {
	message := &sarama.ProducerMessage{
		Topic: fp.kafkaConfig.FileUploadTopic(),
		Key:   sarama.StringEncoder(metadata.FileID),
		Value: sarama.StringEncoder(fp.metadataToJSON(metadata)),
	}
	_, _, err := fp.producer.SendMessage(message)
	return err
}
