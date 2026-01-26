// Package config provides configuration for the application
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Kafka   KafkaConfig
	Logging LoggingConfig
	Service ServiceConfig
	Queue   QueueConfig
	Metrics MetricsConfig
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

	return cfg, nil
}
