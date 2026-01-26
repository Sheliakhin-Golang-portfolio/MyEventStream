// Package obs provides observability functionality including metrics and HTTP endpoints
package obs

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	QueueDepth prometheus.Gauge
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
	}
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
