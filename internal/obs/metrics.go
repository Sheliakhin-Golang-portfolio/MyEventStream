// Package obs provides observability functionality including metrics and HTTP endpoints
package obs

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application.
type Metrics struct {
	QueueDepth           prometheus.Gauge
	EventsIngestedTotal  prometheus.Counter
	EventsProcessedTotal prometheus.Counter
	RetryAttemptsTotal   prometheus.Counter
	DLQMessagesTotal     prometheus.Counter
	RetryExhaustedTotal  prometheus.Counter
}

// NewMetrics creates and initializes a new Metrics instance
// All metrics are registered with the Prometheus default registry
func NewMetrics(serviceName string) *Metrics {
	return &Metrics{
		QueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Current depth of the internal event queue",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		EventsIngestedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_ingested_total",
			Help: "Total number of events ingested from the broker into the internal queue",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		EventsProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events successfully processed by the pipeline",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		RetryAttemptsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts for failed events",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		DLQMessagesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total number of messages sent to the dead-letter queue",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		RetryExhaustedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "retry_exhausted_total",
			Help: "Total number of messages that exhausted all retry attempts",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
	}
}

// IncrementEventsIngested increments the events ingested counter by 1
func (m *Metrics) IncrementEventsIngested() {
	m.EventsIngestedTotal.Inc()
}

// IncrementEventsProcessed increments the events processed counter by 1
func (m *Metrics) IncrementEventsProcessed() {
	m.EventsProcessedTotal.Inc()
}

// IncrementQueueDepth increments the queue depth gauge metric by 1
func (m *Metrics) IncrementQueueDepth() {
	m.QueueDepth.Inc()
}

// DecrementQueueDepth decrements the queue depth gauge metric by 1
func (m *Metrics) DecrementQueueDepth() {
	m.QueueDepth.Dec()
}

// NullifyQueueDepth sets the queue depth gauge metric to 0
func (m *Metrics) NullifyQueueDepth() {
	m.QueueDepth.Set(0)
}

// IncrementRetryAttempts increments the retry attempts counter by 1
func (m *Metrics) IncrementRetryAttempts() {
	m.RetryAttemptsTotal.Inc()
}

// IncrementDLQMessages increments the DLQ messages counter by 1
func (m *Metrics) IncrementDLQMessages() {
	m.DLQMessagesTotal.Inc()
}

// IncrementRetryExhausted increments the retry exhausted counter by 1
func (m *Metrics) IncrementRetryExhausted() {
	m.RetryExhaustedTotal.Inc()
}
