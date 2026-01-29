// Package queue provides a bounded buffered queue for events with backpressure support
package queue

import (
	"context"
	"sync"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/obs"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/types"
)

// Queue represents a bounded buffered channel for events
// When the queue is full, Enqueue blocks, providing backpressure
type Queue struct {
	events  chan *types.Event
	done    chan struct{}
	size    int
	metrics *obs.Metrics
	once    sync.Once
}

// NewQueue creates a new Queue with the specified buffer size
// The queue will block on Enqueue when full, providing backpressure
func NewQueue(size int, metrics *obs.Metrics) *Queue {
	q := &Queue{
		events:  make(chan *types.Event, size),
		done:    make(chan struct{}),
		size:    size,
		metrics: metrics,
	}

	// Initialize queue depth metric to 0
	if metrics != nil {
		metrics.NullifyQueueDepth()
	}

	return q
}

// Enqueue adds an event to the queue
// This operation blocks if the queue is full (backpressure)
// Returns an error if the context is cancelled or the queue is closed
func (q *Queue) Enqueue(ctx context.Context, event *types.Event) error {
	select {
	case q.events <- event:
		// Update metrics after successful enqueue
		if q.metrics != nil {
			q.metrics.IncrementQueueDepth()
			q.metrics.IncrementEventsIngested()
		}
		return nil
	case <-q.done:
		return ErrQueueClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue removes and returns an event from the queue
// This operation blocks if the queue is empty
// Returns an error if the context is cancelled or the queue is closed
func (q *Queue) Dequeue(ctx context.Context) (*types.Event, error) {
	select {
	case event, ok := <-q.events:
		if !ok {
			return nil, ErrQueueClosed
		}
		// Update metrics after successful dequeue
		if q.metrics != nil {
			q.metrics.DecrementQueueDepth()
		}
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Depth returns the current number of events in the queue
func (q *Queue) Depth() int {
	return len(q.events)
}

// Close closes the queue channel gracefully
// After closing, no more events can be enqueued
func (q *Queue) Close() {
	q.once.Do(func() {
		close(q.done)
		close(q.events)
		// Update metrics to 0 after closing
		if q.metrics != nil {
			q.metrics.NullifyQueueDepth()
		}
	})
}

// Errors
var (
	ErrQueueClosed = &QueueError{msg: "queue is closed"}
)

// QueueError represents a queue operation error
type QueueError struct {
	msg string
}

func (e *QueueError) Error() string {
	return e.msg
}
