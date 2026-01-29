// Package dlq provides dead-letter queue functionality for failed messages
package dlq

import (
	"testing"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
	"go.uber.org/zap"
)

func TestNewProducer_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		DLQ: config.DLQConfig{
			Topic:   "test-dlq",
			Brokers: []string{"localhost:9092"},
		},
	}
	logger := zap.NewNop()

	producer, err := NewProducer(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if producer == nil {
		t.Fatal("Expected producer to be non-nil")
	}
	defer producer.Close()

	if producer.topic != "test-dlq" {
		t.Errorf("Expected topic 'test-dlq', got: %s", producer.topic)
	}
}

func TestNewProducer_NilConfig(t *testing.T) {
	logger := zap.NewNop()

	_, err := NewProducer(nil, logger)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}
}

func TestNewProducer_NilLogger(t *testing.T) {
	cfg := &config.Config{
		DLQ: config.DLQConfig{
			Topic:   "test-dlq",
			Brokers: []string{"localhost:9092"},
		},
	}

	_, err := NewProducer(cfg, nil)
	if err == nil {
		t.Fatal("Expected error for nil logger, got nil")
	}
}

func TestPublish_NilEvent(t *testing.T) {
	cfg := &config.Config{
		DLQ: config.DLQConfig{
			Topic:   "test-dlq",
			Brokers: []string{"localhost:9092"},
		},
	}
	logger := zap.NewNop()

	producer, err := NewProducer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Note: We cannot test actual Kafka publish without a running broker
	// This test validates nil event handling
	err = producer.Publish(nil, "test error")
	if err == nil {
		t.Error("Expected error for nil event, got nil")
	}
}

func TestGetHelpers(t *testing.T) {
	event := &types.Event{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Meta: &types.EventMeta{
			Topic:        "test-topic",
			Partition:    5,
			Offset:       123,
			RetryAttempt: 2,
			MaxRetries:   3,
		},
	}

	if getOriginalTopic(event) != "test-topic" {
		t.Errorf("Expected 'test-topic', got: %s", getOriginalTopic(event))
	}
	if getIntFromMeta(event, "partition") != 5 {
		t.Errorf("Expected 5, got: %d", getIntFromMeta(event, "partition"))
	}
	if getOriginalOffset(event) != 123 {
		t.Errorf("Expected 123, got: %d", getOriginalOffset(event))
	}
	if getIntFromMeta(event, "retry_attempts") != 2 {
		t.Errorf("Expected 2, got: %d", getIntFromMeta(event, "retry_attempts"))
	}
}

func TestGetHelpers_NilMeta(t *testing.T) {
	event := &types.Event{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Meta:  nil,
	}

	if getOriginalTopic(event) != "" {
		t.Error("Expected empty string for nil meta")
	}
	if getIntFromMeta(event, "partition") != 0 {
		t.Error("Expected 0 for nil meta")
	}
	if getOriginalOffset(event) != 0 {
		t.Error("Expected 0 for nil meta")
	}
}
