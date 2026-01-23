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
	)

	// Create Kafka consumer
	kafkaConsumer, err := consumer.NewConsumer(cfg, logger.Logger)
	if err != nil {
		logger.Logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
		os.Exit(1)
	}
	defer kafkaConsumer.Close()

	// Create root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			if err != context.Canceled {
				logger.Logger.Error("Consumer error", zap.Error(err))
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

	// Cancel root context to stop consumer
	cancel()

	// Wait for consumer to finish or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Logger.Info("Consumer stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Logger.Warn("Shutdown timeout exceeded, forcing exit")
	}

	logger.Logger.Info("Service exited")
}
