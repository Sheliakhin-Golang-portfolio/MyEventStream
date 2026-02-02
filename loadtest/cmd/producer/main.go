package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var (
	brokers   string
	topic     string
	httpPort  string
	batchSize int
	rate      int
	duration  time.Duration
	payloads  []string
)

func init() {
	// Try to load .env file (optional)
	godotenv.Load()

	flag.StringVar(&brokers, "brokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses (comma-separated)")
	flag.StringVar(&topic, "topic", getEnv("KAFKA_TOPIC", "topic1"), "Kafka topic name")
	flag.StringVar(&httpPort, "http", "", "HTTP server port (e.g., :8081). If set, starts HTTP producer mode")
	flag.IntVar(&batchSize, "batch", 0, "Number of messages to produce (0 = infinite)")
	flag.IntVar(&rate, "rate", 0, "Messages per second (0 = as fast as possible)")
	flag.DurationVar(&duration, "duration", 0, "Duration to run (e.g., 30s, 5m). If set, overrides batch")
	flag.Parse()

	// Get payload files from remaining args or default
	args := flag.Args()
	if len(args) > 0 {
		payloads = args
	} else {
		// Default payload files
		payloads = []string{"payload.json", "payload-2.json", "payload-3.json"}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Parse brokers
	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}

	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	defer writer.Close()

	logger.Info("Kafka producer initialized",
		zap.Strings("brokers", brokerList),
		zap.String("topic", topic),
	)

	// HTTP mode
	if httpPort != "" {
		runHTTPProducer(writer, logger)
		return
	}

	// CLI mode
	runCLIProducer(writer, logger)
}

func runHTTPProducer(writer *kafka.Writer, logger *zap.Logger) {
	logger.Info("Starting HTTP producer server",
		zap.String("port", httpPort),
		zap.String("topic", topic),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", zap.Error(err))
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if len(body) == 0 {
			http.Error(w, "Empty body", http.StatusBadRequest)
			return
		}

		// Produce to Kafka
		msg := kafka.Message{
			Key:   nil,
			Value: body,
			Time:  time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := writer.WriteMessages(ctx, msg); err != nil {
			logger.Error("Failed to produce message", zap.Error(err))
			http.Error(w, "Failed to produce message", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    httpPort,
		Handler: mux,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down HTTP producer server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("HTTP server error", zap.Error(err))
	}
}

func runCLIProducer(writer *kafka.Writer, logger *zap.Logger) {
	// Load payload files
	var payloadData [][]byte
	for _, file := range payloads {
		// Try current directory first, then try loadtest directory relative to executable
		var data []byte
		var err error

		data, err = os.ReadFile(file)
		if err != nil {
			// Try relative to loadtest directory
			loadtestPath := filepath.Join("..", "..", file)
			data, err = os.ReadFile(loadtestPath)
		}

		if err != nil {
			logger.Warn("Failed to read payload file, skipping",
				zap.String("file", file),
				zap.Error(err),
			)
			continue
		}
		payloadData = append(payloadData, data)
	}

	if len(payloadData) == 0 {
		logger.Fatal("No valid payload files found")
	}

	logger.Info("Starting CLI producer",
		zap.Int("batch_size", batchSize),
		zap.Int("rate", rate),
		zap.Duration("duration", duration),
		zap.Int("payload_files", len(payloadData)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutting down producer")
		cancel()
	}()

	// Set duration if specified
	if duration > 0 {
		go func() {
			time.Sleep(duration)
			cancel()
		}()
	}

	// Rate limiter
	var ticker *time.Ticker
	if rate > 0 {
		ticker = time.NewTicker(time.Second / time.Duration(rate))
		defer ticker.Stop()
	}

	var wg sync.WaitGroup
	produced := int64(0)
	var mu sync.Mutex

	// Producer loop
	for {
		if err := ctx.Err(); err != nil {
			logger.Info("Producer stopped",
				zap.Int64("total_produced", produced),
			)
			wg.Wait()
			return
		}

		// Check batch limit
		if batchSize > 0 {
			mu.Lock()
			if produced >= int64(batchSize) {
				mu.Unlock()
				logger.Info("Batch limit reached",
					zap.Int64("total_produced", produced),
				)
				cancel()
				continue
			}
			mu.Unlock()
		}

		// Rate limiting
		if ticker != nil {
			<-ticker.C
		}

		// Select payload (round-robin)
		payload := payloadData[produced%int64(len(payloadData))]

		wg.Add(1)
		go func(payload []byte) {
			defer wg.Done()

			msg := kafka.Message{
				Key:   nil,
				Value: payload,
				Time:  time.Now(),
			}

			writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer writeCancel()

			if err := writer.WriteMessages(writeCtx, msg); err != nil {
				logger.Error("Failed to produce message", zap.Error(err))
				return
			}

			mu.Lock()
			produced++
			current := produced
			mu.Unlock()

			if current%100 == 0 {
				logger.Info("Produced messages", zap.Int64("count", current))
			}
		}(payload)
	}
}
