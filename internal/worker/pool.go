// Package worker provides a fixed-size worker pool for processing events from the queue
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/dlq"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/obs"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/pipeline"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/queue"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/retry"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// CommitFunc is a function type for committing Kafka offsets
type CommitFunc func(kafka.Message)

// Pool represents a fixed-size worker pool that processes events from a queue
type Pool struct {
	workerCount int
	queue       *queue.Queue
	logger      *zap.Logger
	dlqProducer *dlq.Producer
	retryConfig *config.RetryConfig
	commitFunc  CommitFunc
	metrics     *obs.Metrics
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	mu          sync.Mutex
}

// NewPool creates a new worker pool with the specified number of workers
// workerCount must be greater than 0
func NewPool(workerCount int, queue *queue.Queue, logger *zap.Logger, dlqProducer *dlq.Producer, retryConfig *config.RetryConfig, commitFunc CommitFunc, metrics *obs.Metrics) (*Pool, error) {
	if workerCount <= 0 {
		return nil, fmt.Errorf("worker count must be greater than 0, got: %d", workerCount)
	}
	if dlqProducer == nil {
		return nil, fmt.Errorf("dlq producer cannot be nil")
	}
	if commitFunc == nil {
		return nil, fmt.Errorf("commit function cannot be nil")
	}
	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	return &Pool{
		workerCount: workerCount,
		queue:       queue,
		logger:      logger,
		dlqProducer: dlqProducer,
		retryConfig: retryConfig,
		commitFunc:  commitFunc,
		metrics:     metrics,
	}, nil
}

// Start begins processing events from the queue using the worker pool
// It starts N worker goroutines that pull events from the queue and process them
// Returns an error if the pool is already started
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return ErrPoolAlreadyStarted
	}

	p.logger.Info("Starting worker pool",
		zap.Int("workerCount", p.workerCount),
	)

	// Create a context that cancels when the pool is stopped or the main context is cancelled
	p.ctx, p.cancel = context.WithCancel(ctx)

	p.started = true

	// Start worker goroutines
	for i := range p.workerCount {
		p.wg.Add(1)
		go p.worker(i)
	}

	return nil
}

// worker is the main loop for a single worker goroutine
// It pulls events from the queue and processes them until the context is cancelled
func (p *Pool) worker(workerID int) {
	defer p.wg.Done()

	p.logger.Debug("Worker started",
		zap.Int("workerID", workerID),
	)

	for {
		// Check if context is cancelled
		select {
		case <-p.ctx.Done():
			p.logger.Debug("Worker stopping due to context cancellation",
				zap.Int("workerID", workerID),
			)
			return
		default:
		}

		// Dequeue event (blocks if queue is empty)
		event, err := p.queue.Dequeue(p.ctx)
		if err != nil {
			// Handle context cancellation
			if err == context.Canceled || err == context.DeadlineExceeded {
				p.logger.Debug("Worker stopping due to context cancellation during dequeue",
					zap.Int("workerID", workerID),
				)
				return
			}

			// Handle queue closed
			if err == queue.ErrQueueClosed {
				p.logger.Debug("Worker stopping due to queue closed",
					zap.Int("workerID", workerID),
				)
				return
			}

			// Log unexpected errors but continue
			p.logger.Error("Failed to dequeue event",
				zap.Error(err),
				zap.Int("workerID", workerID),
			)
			continue
		}

		// Process event
		p.processEvent(event, workerID)
	}
}

// processEvent processes a single event with retry logic and DLQ handling
func (p *Pool) processEvent(event *types.Event, workerID int) {
	// Check if main context is cancelled before processing
	select {
	case <-p.ctx.Done():
		return
	default:
	}

	// Create Kafka message for commit (from event metadata)
	var kafkaMsg kafka.Message
	if event.Meta != nil {
		kafkaMsg = kafka.Message{
			Topic:     event.Meta.Topic,
			Partition: event.Meta.Partition,
			Offset:    event.Meta.Offset,
		}
	}

	// Wrap pipeline execution in retry logic
	err := retry.DoWithRetry(p.ctx, p.retryConfig, func() error {
		// Increment retry attempt counter
		if event.Meta != nil {
			event.Meta.RetryAttempt++
		}

		// Track retry metric (if not first attempt)
		if event.Meta != nil && event.Meta.RetryAttempt > 1 {
			p.metrics.IncrementRetryAttempts()
		}

		// Log retry attempt
		p.logger.Debug("Processing event",
			zap.Int("workerID", workerID),
			zap.Int("attempt", getAttempt(event)),
			zap.Int("max_attempts", getMaxAttempts(event)),
			zap.String("topic", getTopic(event)),
			zap.Int("partition", getPartition(event)),
			zap.Int64("offset", getOffset(event)),
		)

		// Execute pipeline stages
		payload, err := pipeline.Decode(p.ctx, event.Value)
		if err != nil {
			p.logger.Warn("Pipeline decode failed",
				zap.Error(err),
				zap.Int("workerID", workerID),
				zap.Int("attempt", getAttempt(event)),
				zap.Int("max_attempts", getMaxAttempts(event)),
				zap.String("topic", getTopic(event)),
				zap.Int("partition", getPartition(event)),
				zap.Int64("offset", getOffset(event)),
			)
			return err
		}

		if err := pipeline.Validate(p.ctx, payload); err != nil {
			p.logger.Warn("Pipeline validate failed",
				zap.Error(err),
				zap.Int("workerID", workerID),
				zap.Int("attempt", getAttempt(event)),
				zap.Int("max_attempts", getMaxAttempts(event)),
				zap.String("topic", getTopic(event)),
				zap.Int("partition", getPartition(event)),
				zap.Int64("offset", getOffset(event)),
			)
			return err
		}

		if err := pipeline.Process(p.ctx, payload, p.logger); err != nil {
			p.logger.Warn("Pipeline process failed",
				zap.Error(err),
				zap.Int("workerID", workerID),
				zap.Int("attempt", getAttempt(event)),
				zap.Int("max_attempts", getMaxAttempts(event)),
				zap.String("topic", getTopic(event)),
				zap.Int("partition", getPartition(event)),
				zap.Int64("offset", getOffset(event)),
			)
			return err
		}

		// Success
		p.logger.Info("Event processed successfully",
			zap.Int("workerID", workerID),
			zap.Int("attempt", getAttempt(event)),
			zap.String("topic", getTopic(event)),
			zap.Int("partition", getPartition(event)),
			zap.Int64("offset", getOffset(event)),
		)
		return nil
	})

	// Handle result
	if err == nil {
		p.metrics.IncrementEventsProcessed()
		// Success - commit offset
		if event.Meta != nil {
			p.commitFunc(kafkaMsg)
		}
		return
	}

	// Check if retries were exhausted
	if errors.Is(err, retry.ErrMaxRetriesExceeded) {
		// Track metrics
		p.metrics.IncrementRetryExhausted()

		p.logger.Error("Event processing failed after all retries, sending to DLQ",
			zap.Error(err),
			zap.Int("workerID", workerID),
			zap.Int("retry_attempts", getAttempt(event)),
			zap.String("topic", getTopic(event)),
			zap.Int("partition", getPartition(event)),
			zap.Int64("offset", getOffset(event)),
		)

		// Publish to DLQ
		dlqErr := p.dlqProducer.Publish(event, err.Error())
		if dlqErr != nil {
			p.logger.Error("Failed to publish to DLQ",
				zap.Error(dlqErr),
				zap.Int("workerID", workerID),
				zap.String("topic", getTopic(event)),
				zap.Int("partition", getPartition(event)),
				zap.Int64("offset", getOffset(event)),
			)
			// Even if DLQ fails, we commit to avoid reprocessing
			// This is a trade-off: lose message vs infinite reprocessing
		} else {
			// Track DLQ metric only on successful publish
			p.metrics.IncrementDLQMessages()
		}

		// Commit offset after DLQ (success or failure)
		if event.Meta != nil {
			p.commitFunc(kafkaMsg)
		}
		return
	}

	// Context cancelled or other error
	p.logger.Error("Event processing stopped",
		zap.Error(err),
		zap.Int("workerID", workerID),
		zap.String("topic", getTopic(event)),
		zap.Int("partition", getPartition(event)),
		zap.Int64("offset", getOffset(event)),
	)
}

// Helper functions to safely extract metadata from events
func getTopic(event *types.Event) string {
	if event.Meta == nil {
		return ""
	}
	return event.Meta.Topic
}

func getPartition(event *types.Event) int {
	if event.Meta == nil {
		return -1
	}
	return event.Meta.Partition
}

func getOffset(event *types.Event) int64 {
	if event.Meta == nil {
		return -1
	}
	return event.Meta.Offset
}

func getAttempt(event *types.Event) int {
	if event.Meta == nil {
		return 0
	}
	return event.Meta.RetryAttempt
}

func getMaxAttempts(event *types.Event) int {
	if event.Meta == nil {
		return 0
	}
	return event.Meta.MaxRetries + 1 // total attempts = max retries + 1 initial attempt
}

// Stop gracefully stops the worker pool
// It cancels the internal context and waits for all workers to finish
// This ensures in-flight events are processed before shutdown
func (p *Pool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}
	p.started = false

	p.logger.Info("Stopping worker pool",
		zap.Int("workerCount", p.workerCount),
	)

	// Cancel context if it exists
	if p.cancel != nil {
		p.cancel()
	}

	// Wait for all workers to finish
	p.wg.Wait()

	p.logger.Info("Worker pool stopped",
		zap.Int("workerCount", p.workerCount),
	)

	// Clear context and cancel function to make it obviously invalid
	p.ctx = nil
	p.cancel = nil

	return nil
}

// Errors
var (
	ErrPoolAlreadyStarted = &PoolError{msg: "worker pool is already started"}
)

// PoolError represents a worker pool operation error
type PoolError struct {
	msg string
}

func (e *PoolError) Error() string {
	return e.msg
}
