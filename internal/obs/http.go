// Package obs provides observability functionality including metrics and HTTP endpoints
package obs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// StartMetricsServer starts an HTTP server that exposes Prometheus metrics
// The server listens on the specified port and exposes the /metrics endpoint
// It respects context cancellation for graceful shutdown
func StartMetricsServer(ctx context.Context, port string, logger *zap.Logger) error {
	// Validate port
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("invalid port: %s", port)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Starting metrics server",
			zap.String("address", server.Addr),
			zap.String("endpoint", "/metrics"),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down metrics server")
		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error shutting down metrics server", zap.Error(err))
			return fmt.Errorf("error shutting down metrics server: %w", err)
		}
		logger.Info("Metrics server stopped gracefully")
		return nil
	case err := <-serverErr:
		return fmt.Errorf("metrics server error: %w", err)
	}
}
