// Package types defines shared types used across the application
package types

// Event represents a message event with key and value
// Both Key and Value are byte slices to handle arbitrary data formats
type Event struct {
	Key   []byte
	Value []byte
	// Meta contains optional metadata about the event (Kafka metadata, retry info)
	Meta *EventMeta
}

// EventMeta contains metadata about an event for tracking and processing
type EventMeta struct {
	// Kafka message metadata
	Topic     string
	Partition int
	Offset    int64

	// Retry tracking
	RetryAttempt int
	MaxRetries   int
}
