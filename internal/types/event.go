// Package types defines shared types used across the application
package types

// Event represents a message event with key and value
// Both Key and Value are byte slices to handle arbitrary data formats
type Event struct {
	Key   []byte
	Value []byte
}
