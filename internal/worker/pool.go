// Package worker provides a fixed-size worker pool for processing events from the queue
package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/pipeline"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/queue"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
	"go.uber.org/zap"
)

// Pool represents a fixed-size worker pool that processes events from a queue
type Pool struct {
	workerCount int
	queue       *queue.Queue
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	mu          sync.Mutex
}

// NewPool creates a new worker pool with the specified number of workers
// workerCount must be greater than 0
func NewPool(workerCount int, queue *queue.Queue, logger *zap.Logger) (*Pool, error) {
	if workerCount <= 0 {
		return nil, fmt.Errorf("worker count must be greater than 0, got: %d", workerCount)
	}

	return &Pool{
		workerCount: workerCount,
		queue:       queue,
		logger:      logger,
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

	// Create a context that cancels when the pool is stopped
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.started = true

	// Start worker goroutines
	for i := range p.workerCount {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	return nil
}

// worker is the main loop for a single worker goroutine
// It pulls events from the queue and processes them until the context is cancelled
func (p *Pool) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	p.logger.Debug("Worker started",
		zap.Int("workerID", workerID),
	)

	// Create a combined context that cancels when either ctx or p.ctx cancels
	// This ensures Dequeue can be interrupted by either context
	combinedCtx, combinedCancel := p.combineContexts(ctx)
	defer combinedCancel()

	for {
		// Check if context is cancelled
		select {
		case <-combinedCtx.Done():
			// Determine which context cancelled for logging
			if ctx.Err() != nil {
				p.logger.Debug("Worker stopping due to context cancellation",
					zap.Int("workerID", workerID),
				)
			} else if p.ctx.Err() != nil {
				p.logger.Debug("Worker stopping due to pool cancellation",
					zap.Int("workerID", workerID),
				)
			}
			return
		default:
		}

		// Dequeue event (blocks if queue is empty)
		event, err := p.queue.Dequeue(combinedCtx)
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

		// Process event (placeholder for Stage 0.4)
		p.processEvent(ctx, event, workerID)
	}
}

// combineContexts creates a context that cancels when either ctx1 or pool's context cancels
// Returns the combined context and a cancel function that should be called to clean up
func (p *Pool) combineContexts(ctx1 context.Context) (context.Context, context.CancelFunc) {
	combinedCtx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-ctx1.Done():
			cancel()
		case <-p.ctx.Done():
			cancel()
		case <-combinedCtx.Done():
			// Combined context was cancelled externally, nothing to do
		}
	}()

	return combinedCtx, cancel
}

// processEvent processes a single event
// For Stage 0.4, this is a placeholder that logs event metadata
func (p *Pool) processEvent(ctx context.Context, event *types.Event, workerID int) {
	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		return
	case <-p.ctx.Done():
		return
	default:
	}

	payload, err := pipeline.Decode(ctx, event.Value)
	if err != nil {
		p.logger.Error("Pipeline decode failed",
			zap.Error(err),
			zap.Int("workerID", workerID),
			zap.Int("keyLength", len(event.Key)),
			zap.Int("valueLength", len(event.Value)),
			zap.Int("queueDepth", p.queue.Depth()),
		)
		return
	}

	if err := pipeline.Validate(ctx, payload); err != nil {
		p.logger.Error("Pipeline validate failed",
			zap.Error(err),
			zap.Int("workerID", workerID),
			zap.Int("keyLength", len(event.Key)),
			zap.Int("valueLength", len(event.Value)),
			zap.Int("queueDepth", p.queue.Depth()),
		)
		return
	}

	if err := pipeline.Process(ctx, payload, p.logger); err != nil {
		p.logger.Error("Pipeline process failed",
			zap.Error(err),
			zap.Int("workerID", workerID),
			zap.Int("keyLength", len(event.Key)),
			zap.Int("valueLength", len(event.Value)),
			zap.Int("queueDepth", p.queue.Depth()),
		)
		return
	}
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
