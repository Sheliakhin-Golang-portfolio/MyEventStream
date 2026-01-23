package consumer

import (
	"context"
	"fmt"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer represents a Kafka consumer
type Consumer struct {
	reader *kafka.Reader
	logger *zap.Logger
}

// NewConsumer creates a new Kafka consumer instance
func NewConsumer(cfg *config.Config, logger *zap.Logger) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topic,
		GroupID:  cfg.Kafka.GroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}, nil
}

// Start begins consuming messages from Kafka
// It respects context cancellation for graceful shutdown
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer",
		zap.Strings("brokers", c.reader.Config().Brokers),
		zap.String("topic", c.reader.Config().Topic),
		zap.String("groupID", c.reader.Config().GroupID),
	)

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			return ctx.Err()
		default:
		}

		// Read message with context
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation
			if err == context.Canceled || err == context.DeadlineExceeded {
				c.logger.Info("Consumer stopped due to context cancellation")
				return err
			}
			// Log error and continue (network errors are transient)
			c.logger.Error("Failed to read message from Kafka",
				zap.Error(err),
			)
			// Continue to retry
			continue
		}

		// Log message metadata only (not content)
		c.logger.Info("Received message",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.Int("keyLength", len(msg.Key)),
			zap.Int("valueLength", len(msg.Value)),
			zap.Time("timestamp", msg.Time),
		)

		// For now, we rely on the Reader's automatic commit behavior,
		// but we can explicitly commit if needed later.
	}
}

// Close closes the Kafka consumer and releases resources
func (c *Consumer) Close() error {
	if c.reader != nil {
		c.logger.Info("Closing Kafka consumer")
		if err := c.reader.Close(); err != nil {
			return fmt.Errorf("failed to close Kafka reader: %w", err)
		}
	}
	return nil
}
