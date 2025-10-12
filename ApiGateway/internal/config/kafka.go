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
)

type KafkaConfig interface {
	Brokers() []string
	Topic() string
	GroupID() string
	Workers() int
	ConsumerTimeout() int
}

type kafkaConfig struct {
	brokers         []string
	topic           string
	groupID         string
	workers         int
	consumerTimeout int
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

	return &kafkaConfig{
		brokers:         brokers,
		topic:           topic,
		groupID:         groupID,
		workers:         workers,
		consumerTimeout: timeout,
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
