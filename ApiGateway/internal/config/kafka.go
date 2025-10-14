package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

var _ KafkaConfig = (*kafkaConfig)(nil)

const (
	kafkaBrokersEnvName         = "KAFKA_BROKERS"
	kafkaTopicEnvName           = "KAFKA_TOPIC"
	kafkaGroupIDEnvName         = "KAFKA_GROUP_ID"
	kafkaWorkersEnvName         = "KAFKA_WORKERS"
	kafkaConsumerTimeoutEnvName = "KAFKA_CONSUMER_TIMEOUT"

	kafkaFileUploadTopicEnvName    = "KAFKA_FILE_UPLOAD_TOPIC"
	kafkaFileProcessedTopicEnvName = "KAFKA_FILE_PROCESSED_TOPIC"
	kafkaBatchSizeEnvName          = "KAFKA_BATCH_SIZE"
	kafkaLingerMsEnvName           = "KAFKA_LINGER_MS"
)

type KafkaConfig interface {
	Brokers() []string
	Topic() string
	GroupID() string
	Workers() int
	ConsumerTimeout() int

	FileUploadTopic() string
	FileProcessedTopic() string
	BatchSize() int
	LingerMs() int
}

type kafkaConfig struct {
	brokers         []string
	topic           string
	groupID         string
	workers         int
	consumerTimeout int

	fileUploadTopic    string
	fileProcessedTopic string
	batchSize          int
	lingerMs           int
}

func NewKafkaConfig() (KafkaConfig, error) {
	brokersStr := os.Getenv(kafkaBrokersEnvName)
	if len(brokersStr) == 0 {
		return nil, errors.New("kafka brokers not found")
	}

	brokers := strings.Split(brokersStr, ",")
	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}

	topic := os.Getenv(kafkaTopicEnvName)
	if len(topic) == 0 {
		return nil, errors.New("kafka topic not found")
	}

	groupID := os.Getenv(kafkaGroupIDEnvName)
	if len(groupID) == 0 {
		groupID = "api-gateway"
	}

	workersStr := os.Getenv(kafkaWorkersEnvName)
	workers := 3
	if workersStr != "" {
		var err error
		workers, err = strconv.Atoi(workersStr)
		if err != nil || workers <= 0 {
			return nil, errors.New("invalid kafka workers count")
		}
	}

	timeoutStr := os.Getenv(kafkaConsumerTimeoutEnvName)
	timeout := 30
	if timeoutStr != "" {
		var err error
		timeout, err = strconv.Atoi(timeoutStr)
		if err != nil || timeout <= 0 {
			return nil, errors.New("invalid kafka consumer timeout")
		}
	}

	fileUploadTopic := os.Getenv(kafkaFileUploadTopicEnvName)
	if fileUploadTopic == "" {
		fileUploadTopic = "file-upload-topic"
	}

	fileProcessedTopic := os.Getenv(kafkaFileProcessedTopicEnvName)
	if fileProcessedTopic == "" {
		fileProcessedTopic = "file-processed-topic"
	}

	batchSizeStr := os.Getenv(kafkaBatchSizeEnvName)
	batchSize := 16384
	if batchSizeStr != "" {
		var err error
		batchSize, err = strconv.Atoi(batchSizeStr)
		if err != nil || batchSize <= 0 {
			return nil, errors.New("invalid kafka batch size")
		}
	}

	lingerMsStr := os.Getenv(kafkaLingerMsEnvName)
	lingerMs := 20
	if lingerMsStr != "" {
		var err error
		lingerMs, err = strconv.Atoi(lingerMsStr)
		if err != nil || lingerMs < 0 {
			return nil, errors.New("invalid kafka linger ms")
		}
	}

	return &kafkaConfig{
		brokers:         brokers,
		topic:           topic,
		groupID:         groupID,
		workers:         workers,
		consumerTimeout: timeout,

		fileUploadTopic:    fileUploadTopic,
		fileProcessedTopic: fileProcessedTopic,
		batchSize:          batchSize,
		lingerMs:           lingerMs,
	}, nil
}

func (cfg *kafkaConfig) Brokers() []string {
	return cfg.brokers
}

func (cfg *kafkaConfig) Topic() string {
	return cfg.topic
}

func (cfg *kafkaConfig) GroupID() string {
	return cfg.groupID
}

func (cfg *kafkaConfig) Workers() int {
	return cfg.workers
}

func (cfg *kafkaConfig) ConsumerTimeout() int {
	return cfg.consumerTimeout
}

func (cfg *kafkaConfig) FileUploadTopic() string {
	return cfg.fileUploadTopic
}

func (cfg *kafkaConfig) FileProcessedTopic() string {
	return cfg.fileProcessedTopic
}

func (cfg *kafkaConfig) BatchSize() int {
	return cfg.batchSize
}

func (cfg *kafkaConfig) LingerMs() int {
	return cfg.lingerMs
}
