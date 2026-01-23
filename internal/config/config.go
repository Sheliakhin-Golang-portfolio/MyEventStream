// Package config provides configuration for the application
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Kafka   KafkaConfig
	Logging LoggingConfig
	Service ServiceConfig
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

	return cfg, nil
}
