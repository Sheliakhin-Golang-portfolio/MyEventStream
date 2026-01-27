// Package main is the entry point for MyEventStream.
// MyEventStream is a production-style event processing service that consumes events
// from a message broker, applies controlled concurrency and backpressure, and
// processes messages reliably with retries and a dead-letter queue.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/config"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/consumer"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/logger"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/obs"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/queue"
	"github.com/Sheliakhin-Golang-portfolio/MyEventStream/internal/worker"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v\n", err)
	}
	defer logger.Sync()

	logger.Logger.Info("Starting MyEventStream service",
		zap.String("serviceName", cfg.Service.Name),
		zap.Int("queueBufferSize", cfg.Queue.BufferSize),
		zap.Int("workerPoolSize", cfg.WorkerPool.Size),
		zap.String("metricsPort", cfg.Metrics.Port),
	)

	// Initialize metrics
	metrics := obs.NewMetrics(cfg.Service.Name)

	// Create queue with metrics
	eventQueue := queue.NewQueue(cfg.Queue.BufferSize, metrics)
	defer eventQueue.Close()

	// Create worker pool
	workerPool, err := worker.NewPool(cfg.WorkerPool.Size, eventQueue, logger.Logger)
	if err != nil {
		logger.Logger.Fatal("Failed to create worker pool", zap.Error(err))
		os.Exit(1)
	}
	defer workerPool.Stop()

	// Create Kafka consumer with queue
	kafkaConsumer, err := consumer.NewConsumer(cfg, logger.Logger, eventQueue)
	if err != nil {
		logger.Logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
		os.Exit(1)
	}
	defer kafkaConsumer.Close()

	// Create root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics HTTP server in goroutine
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := obs.StartMetricsServer(ctx, cfg.Metrics.Port, logger.Logger); err != nil {
			if err != context.Canceled {
				logger.Logger.Error("Metrics server error", zap.Error(err))
			}
		}
	})

	// Start consumer in goroutine
	wg.Go(func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			if err != context.Canceled {
				logger.Logger.Error("Consumer error", zap.Error(err))
			}
		}
	})

	// Start worker pool in goroutine
	wg.Go(func() {
		if err := workerPool.Start(ctx); err != nil {
			if err != context.Canceled {
				logger.Logger.Error("Worker pool error", zap.Error(err))
			}
		}
	})

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down service...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel root context to stop consumer and worker pool
	cancel()

	// Stop worker pool gracefully (finishes in-flight work)
	if err := workerPool.Stop(); err != nil {
		logger.Logger.Error("Error stopping worker pool", zap.Error(err))
	}

	// Wait for all services to finish or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Logger.Info("All services stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Logger.Warn("Shutdown timeout exceeded, forcing exit")
	}

	logger.Logger.Info("Service exited")
}
