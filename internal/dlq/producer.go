// Package dlq provides dead-letter queue functionality for failed messages
package dlq

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer handles publishing failed messages to the dead-letter queue
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

// NewProducer creates a new DLQ producer
func NewProducer(cfg *config.Config, logger *zap.Logger) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.DLQ.Brokers...),
		Topic:        cfg.DLQ.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	return &Producer{
		writer: writer,
		logger: logger,
		topic:  cfg.DLQ.Topic,
	}, nil
}

// Publish sends a failed message to the DLQ with metadata
func (p *Producer) Publish(event *types.Event, errorMsg string) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Build Kafka headers with metadata
	headers := []kafka.Header{
		{Key: "error_message", Value: []byte(errorMsg)},
		{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
	}

	// Add event metadata if available
	if event.Meta != nil {
		headers = append(headers,
			kafka.Header{Key: "original_topic", Value: []byte(event.Meta.Topic)},
			kafka.Header{Key: "original_partition", Value: []byte(strconv.Itoa(event.Meta.Partition))},
			kafka.Header{Key: "original_offset", Value: []byte(strconv.FormatInt(event.Meta.Offset, 10))},
			kafka.Header{Key: "retry_attempts", Value: []byte(strconv.Itoa(event.Meta.RetryAttempt))},
			kafka.Header{Key: "max_retries", Value: []byte(strconv.Itoa(event.Meta.MaxRetries))},
		)
	}

	// Create Kafka message for DLQ
	msg := kafka.Message{
		Key:     event.Key,
		Value:   event.Value,
		Headers: headers,
		Time:    time.Now(),
	}

	// Attempt to write to DLQ with a small number of retries
	maxDLQAttempts := 3
	var lastErr error

	for attempt := range maxDLQAttempts {
		// We use a context that will never be cancelled to ensure that the DLQ publish will not be interrupted by context cancellation or timeout
		// We rely on the kafka-go writer to handle the timeouts.
		lastErr = p.writer.WriteMessages(context.Background(), msg)
		if lastErr == nil {
			p.logger.Info("Message published to DLQ",
				zap.String("dlq_topic", p.topic),
				zap.String("original_topic", getOriginalTopic(event)),
				zap.Int("original_partition", getIntFromMeta(event, "partition")),
				zap.Int64("original_offset", getOriginalOffset(event)),
				zap.Int("retry_attempts", getIntFromMeta(event, "retry_attempts")),
				zap.String("error", errorMsg),
			)
			return nil
		}

		p.logger.Warn("Failed to publish to DLQ, will retry",
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", maxDLQAttempts),
			zap.Error(lastErr),
		)

		// Short delay before retry
		if attempt < maxDLQAttempts-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// All DLQ publish attempts failed
	p.logger.Error("Failed to publish message to DLQ after all attempts",
		zap.String("dlq_topic", p.topic),
		zap.String("original_topic", getOriginalTopic(event)),
		zap.Int("max_attempts", maxDLQAttempts),
		zap.Error(lastErr),
	)

	return fmt.Errorf("failed to publish to DLQ after %d attempts: %w", maxDLQAttempts, lastErr)
}

// Close closes the DLQ producer and releases resources
func (p *Producer) Close() error {
	if p.writer != nil {
		p.logger.Info("Closing DLQ producer")
		return p.writer.Close()
	}
	return nil
}

// Helper functions to extract topic safely
func getOriginalTopic(event *types.Event) string {
	if event.Meta == nil {
		return ""
	}

	return event.Meta.Topic
}

// Helper functions to extract partition or retry attempts safely
func getIntFromMeta(event *types.Event, field string) int {
	if event.Meta == nil {
		return 0
	}
	switch field {
	case "partition":
		return event.Meta.Partition
	case "retry_attempts":
		return event.Meta.RetryAttempt
	default:
		return 0
	}
}

// Helper functions to extract offset safely
func getOriginalOffset(event *types.Event) int64 {
	if event.Meta == nil {
		return 0
	}

	return event.Meta.Offset
}
