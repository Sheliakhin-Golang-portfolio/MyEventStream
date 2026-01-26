package consumer

import (
	"context"
	"fmt"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/queue"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer represents a Kafka consumer
type Consumer struct {
	reader *kafka.Reader
	logger *zap.Logger
	queue  *queue.Queue
}

// NewConsumer creates a new Kafka consumer instance
func NewConsumer(cfg *config.Config, logger *zap.Logger, queue *queue.Queue) (*Consumer, error) {
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
		queue:  queue,
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

		// Create event from Kafka message
		event := &types.Event{
			Key:   msg.Key,
			Value: msg.Value,
		}

		// Enqueue event (blocks if queue is full - backpressure)
		if err := c.queue.Enqueue(ctx, event); err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				c.logger.Info("Consumer stopped due to context cancellation during enqueue")
				return err
			}
			c.logger.Error("Failed to enqueue event",
				zap.Error(err),
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
			)
			// Continue to retry
			continue
		}

		// Log message metadata only (not content)
		c.logger.Info("Enqueued message",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.Int("keyLength", len(msg.Key)),
			zap.Int("valueLength", len(msg.Value)),
			zap.Time("timestamp", msg.Time),
			zap.Int("queueDepth", c.queue.Depth()),
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
