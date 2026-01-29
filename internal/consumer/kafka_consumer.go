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
	reader     *kafka.Reader
	logger     *zap.Logger
	queue      *queue.Queue
	commitChan chan kafka.Message
	maxRetries int
}

// NewConsumer creates a new Kafka consumer instance
func NewConsumer(cfg *config.Config, logger *zap.Logger, queue *queue.Queue) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		Topic:          cfg.Kafka.Topic,
		GroupID:        cfg.Kafka.GroupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 0,    // Manual commit only
	})

	return &Consumer{
		reader:     reader,
		logger:     logger,
		queue:      queue,
		commitChan: make(chan kafka.Message, 100),
		maxRetries: cfg.Retry.MaxAttempts,
	}, nil
}

// CommitMessage queues a message for commit (called by worker after success/DLQ)
func (c *Consumer) CommitMessage(msg kafka.Message) {
	select {
	case c.commitChan <- msg:
		// Message queued for commit
	default:
		// Commit channel full - log warning
		c.logger.Warn("Commit channel full, message may be re-processed",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)
	}
}

// Start begins consuming messages from Kafka
// It respects context cancellation for graceful shutdown
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer",
		zap.Strings("brokers", c.reader.Config().Brokers),
		zap.String("topic", c.reader.Config().Topic),
		zap.String("groupID", c.reader.Config().GroupID),
	)

	commitDone := make(chan struct{})
	go c.commitLoop(commitDone)

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			close(c.commitChan)
			<-commitDone
			return ctx.Err()
		default:
		}

		// Fetch message with context (does NOT auto-commit)
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation
			if err == context.Canceled || err == context.DeadlineExceeded {
				c.logger.Info("Consumer stopped due to context cancellation")
				close(c.commitChan)
				<-commitDone
				return err
			}
			// Log error and continue (network errors are transient)
			c.logger.Error("Failed to fetch message from Kafka",
				zap.Error(err),
			)
			// Continue to retry
			continue
		}

		// Create event from Kafka message with metadata
		event := &types.Event{
			Key:   msg.Key,
			Value: msg.Value,
			Meta: &types.EventMeta{
				Topic:      msg.Topic,
				Partition:  msg.Partition,
				Offset:     msg.Offset,
				MaxRetries: c.maxRetries,
			},
		}

		// Enqueue event (blocks if queue is full - backpressure)
		if err := c.queue.Enqueue(ctx, event); err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				c.logger.Info("Consumer stopped due to context cancellation during enqueue")
				close(c.commitChan)
				<-commitDone
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

		// Note: Offset commit happens after worker processes (success or DLQ)
		// The worker will call consumer.CommitMessage() when done
	}
}

// commitLoop runs in a goroutine and commits offsets as they are received
func (c *Consumer) commitLoop(done chan struct{}) {
	// Close the done channel when the function returns to signal that the commit loop has stopped
	defer close(done)

	// We use a separate context to ensure that the commit channel will be drained before the consumer is closed
	commitCtx := context.Background()

	// Use a range loop to ensure that the commit channel will be drained before the consumer is closed
	for msg := range c.commitChan {
		if err := c.reader.CommitMessages(commitCtx, msg); err != nil {
			c.logger.Error("Failed to commit offset",
				zap.Error(err),
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
			)
		} else {
			c.logger.Debug("Committed offset",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
			)
		}
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
