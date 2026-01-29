// Package config provides configuration for the application
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Kafka      KafkaConfig
	Logging    LoggingConfig
	Service    ServiceConfig
	Queue      QueueConfig
	Metrics    MetricsConfig
	WorkerPool WorkerPoolConfig
	Retry      RetryConfig
	DLQ        DLQConfig
}

// KafkaConfig holds Kafka connection settings
type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string
}

// ServiceConfig holds service settings
type ServiceConfig struct {
	Name string
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	BufferSize int
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Port string
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	Size int
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int
	// Base delay in milliseconds
	BaseDelayMs time.Duration
	// Max delay in milliseconds
	MaxDelayMs time.Duration
	Multiplier float64
}

// DLQConfig holds dead-letter queue configuration
type DLQConfig struct {
	Topic   string
	Brokers []string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional)
	godotenv.Load()

	cfg := &Config{}

	// Kafka configuration
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS is required")
	}
	// Parse comma-separated brokers
	brokers := strings.Split(kafkaBrokers, ",")
	cfg.Kafka.Brokers = make([]string, 0, len(brokers))
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			cfg.Kafka.Brokers = append(cfg.Kafka.Brokers, broker)
		}
	}
	if len(cfg.Kafka.Brokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS must contain at least one valid broker address")
	}

	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		return nil, fmt.Errorf("KAFKA_TOPIC is required")
	}
	cfg.Kafka.Topic = kafkaTopic

	kafkaGroupID := os.Getenv("KAFKA_GROUP_ID")
	if kafkaGroupID == "" {
		kafkaGroupID = "myeventstream-consumer" // default
	}
	cfg.Kafka.GroupID = kafkaGroupID

	// Logging configuration
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // default level
	}
	cfg.Logging.Level = logLevel

	// Service configuration
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "myeventstream" // default
	}
	cfg.Service.Name = serviceName

	// Queue configuration
	queueBufferSizeStr := os.Getenv("QUEUE_BUFFER_SIZE")
	if queueBufferSizeStr == "" {
		queueBufferSizeStr = "1000" // default
	}
	queueBufferSize, err := strconv.Atoi(queueBufferSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid QUEUE_BUFFER_SIZE: %w", err)
	}
	if queueBufferSize <= 0 {
		return nil, fmt.Errorf("QUEUE_BUFFER_SIZE must be greater than 0, got: %d", queueBufferSize)
	}
	cfg.Queue.BufferSize = queueBufferSize

	// Metrics configuration
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8080" // default
	}
	cfg.Metrics.Port = metricsPort

	// Worker pool configuration
	workerPoolSizeStr := os.Getenv("WORKER_POOL_SIZE")
	if workerPoolSizeStr == "" {
		workerPoolSizeStr = "10" // default
	}
	workerPoolSize, err := strconv.Atoi(workerPoolSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid WORKER_POOL_SIZE: %w", err)
	}
	if workerPoolSize <= 0 {
		return nil, fmt.Errorf("WORKER_POOL_SIZE must be greater than 0, got: %d", workerPoolSize)
	}
	cfg.WorkerPool.Size = workerPoolSize

	// Retry configuration
	retryMaxAttemptsStr := os.Getenv("RETRY_MAX_ATTEMPTS")
	if retryMaxAttemptsStr == "" {
		retryMaxAttemptsStr = "3" // default
	}
	retryMaxAttempts, err := strconv.Atoi(retryMaxAttemptsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_MAX_ATTEMPTS: %w", err)
	}
	if retryMaxAttempts < 0 {
		return nil, fmt.Errorf("RETRY_MAX_ATTEMPTS must be >= 0, got: %d", retryMaxAttempts)
	}
	cfg.Retry.MaxAttempts = retryMaxAttempts

	retryBaseDelayMsStr := os.Getenv("RETRY_BASE_DELAY_MS")
	if retryBaseDelayMsStr == "" {
		retryBaseDelayMsStr = "100" // default 100ms
	}
	retryBaseDelay, err := strconv.Atoi(retryBaseDelayMsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_BASE_DELAY_MS: %w", err)
	}
	if retryBaseDelay <= 0 {
		return nil, fmt.Errorf("RETRY_BASE_DELAY_MS must be > 0, got: %d", retryBaseDelay)
	}
	cfg.Retry.BaseDelayMs = time.Duration(retryBaseDelay) * time.Millisecond

	retryMaxDelayMsStr := os.Getenv("RETRY_MAX_DELAY_MS")
	if retryMaxDelayMsStr == "" {
		retryMaxDelayMsStr = "10000" // default 10s
	}
	retryMaxDelay, err := strconv.Atoi(retryMaxDelayMsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_MAX_DELAY_MS: %w", err)
	}
	if retryMaxDelay <= 0 {
		return nil, fmt.Errorf("RETRY_MAX_DELAY_MS must be > 0, got: %d", retryMaxDelay)
	}
	cfg.Retry.MaxDelayMs = time.Duration(retryMaxDelay) * time.Millisecond

	retryMultiplierStr := os.Getenv("RETRY_MULTIPLIER")
	if retryMultiplierStr == "" {
		retryMultiplierStr = "2.0" // default exponential backoff
	}
	retryMultiplier, err := strconv.ParseFloat(retryMultiplierStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_MULTIPLIER: %w", err)
	}
	if retryMultiplier <= 0 {
		return nil, fmt.Errorf("RETRY_MULTIPLIER must be > 0, got: %f", retryMultiplier)
	}
	cfg.Retry.Multiplier = retryMultiplier

	// DLQ configuration
	dlqTopic := os.Getenv("DLQ_TOPIC")
	if dlqTopic == "" {
		dlqTopic = "myeventstream-dlq" // default
	}
	cfg.DLQ.Topic = dlqTopic

	// DLQ brokers (reuse main Kafka brokers if not specified)
	dlqBrokers := os.Getenv("DLQ_BROKERS")
	if dlqBrokers == "" {
		cfg.DLQ.Brokers = cfg.Kafka.Brokers
	} else {
		brokers := strings.Split(dlqBrokers, ",")
		cfg.DLQ.Brokers = make([]string, 0, len(brokers))
		for _, broker := range brokers {
			broker = strings.TrimSpace(broker)
			if broker != "" {
				cfg.DLQ.Brokers = append(cfg.DLQ.Brokers, broker)
			}
		}
		if len(cfg.DLQ.Brokers) == 0 {
			return nil, fmt.Errorf("DLQ_BROKERS must contain at least one valid broker address")
		}
	}

	return cfg, nil
}
