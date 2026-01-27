# Changelog

All notable changes to the MyEventStream project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.4.0] - Stage 0.4: Worker Pool Implementation

### Added

#### Worker Pool Package
- **`internal/worker/pool.go`**: Fixed-size worker pool implementation
  - `Pool` struct with configurable worker count
  - `NewPool(workerCount int, queue *queue.Queue, logger *zap.Logger)` constructor
  - `Start(ctx context.Context) error` - starts worker goroutines and begins processing
  - `Stop() error` - gracefully stops all workers and waits for in-flight work to complete
  - `worker(ctx context.Context, workerID int)` - main worker loop that dequeues and processes events
  - `processEvent(ctx context.Context, event *types.Event, workerID int)` - placeholder event processing
  - Thread-safe implementation with `sync.Mutex` for state protection
  - Uses `sync.WaitGroup` to coordinate worker lifecycle
  - Context-based cancellation for graceful shutdown
  - Combined context handling to support both root context and pool-level cancellation
  - Handles `context.Canceled`, `context.DeadlineExceeded`, and `queue.ErrQueueClosed` errors gracefully
  - Worker ID tracking for observability and debugging
  - Prevents double-start with `started` flag and `ErrPoolAlreadyStarted` error

#### Configuration Updates
- **`internal/config/config.go`**: Worker pool configuration
  - `WorkerPoolConfig` struct with `Size int` field
  - `WorkerPool` field added to `Config` struct
  - `WORKER_POOL_SIZE` environment variable support (default: 10)
  - Validation: `WORKER_POOL_SIZE` must be > 0

#### Main Application Updates
- **`cmd/main.go`**: Worker pool integration
  - Worker pool creation with configurable size from environment
  - Worker pool started in goroutine using `sync.WaitGroup.Go()`
  - Enhanced graceful shutdown sequence:
    1. Context cancellation signals all components
    2. Worker pool stops gracefully (finishes in-flight events)
    3. Queue closure
    4. Consumer closure
    5. Metrics server shutdown
  - Improved error handling for worker pool lifecycle
  - Logging includes worker pool size in startup information

#### Documentation Updates
- **`docs/CONFIGURATION.md`**: Added `WORKER_POOL_SIZE` documentation
  - Default value: 10
  - Description: Number of concurrent workers
  - Usage context and tuning guidance

### Changed

- **`cmd/main.go`**:
  - Enhanced graceful shutdown to include worker pool
  - Worker pool stops before queue closure to ensure in-flight events complete
  - Improved shutdown timeout handling (30 seconds)
  - Better goroutine coordination using `sync.WaitGroup`

- **`.env.example`**:
  - Added `WORKER_POOL_SIZE` with default value (10)
  - Added inline documentation for worker pool configuration

### Technical Details

- Worker pool implements fixed-size concurrency model
- Workers block on `queue.Dequeue()` when queue is empty (expected behavior)
- Graceful shutdown ensures no messages are dropped during service termination
- Context cancellation propagates to all workers simultaneously
- Worker pool respects both root context (for shutdown) and internal context (for pool lifecycle)
- Event processing is currently a placeholder that logs metadata (key length, value length, queue depth)
- Worker ID is included in all processing logs for observability
- Pool state is protected by mutex to prevent race conditions during start/stop operations
- All workers complete their current event before shutdown completes

---

## [0.3.0] - Stage 0.3: Queue & Backpressure Integration

### Added

#### Event Type
- **`internal/types/event.go`**: Shared event type definition
  - `Event` struct with `Key []byte` and `Value []byte` fields
  - Used across consumer, queue, and future processing stages
  - Supports arbitrary data formats

#### Queue Package
- **`internal/queue/queue.go`**: Bounded buffered queue implementation
  - `Queue` struct with bounded channel for events
  - `NewQueue(size int, metrics *obs.Metrics)` constructor
  - `Enqueue(ctx context.Context, event *types.Event) error` - blocking enqueue with backpressure
  - `Dequeue(ctx context.Context) (*types.Event, error)` - blocking dequeue
  - `Depth() int` - returns current queue depth
  - `Close()` - graceful channel closure using `sync.Once`
  - Backpressure: blocks when queue is full, preventing unbounded memory growth
  - Thread-safe implementation with `done` channel for closure signaling
  - Automatic metrics updates on enqueue/dequeue operations

#### Observability Package
- **`internal/obs/metrics.go`**: Prometheus metrics implementation
  - `Metrics` struct with `QueueDepth` gauge metric
  - `NewMetrics(serviceName string)` constructor
  - `IncrementQueueDepth()` - increments queue depth metric
  - `DecrementQueueDepth()` - decrements queue depth metric
  - `NullifyQueueDepth()` - sets queue depth to 0
  - Metrics registered with Prometheus default registry
  - Service name label for metric identification

- **`internal/obs/http.go`**: HTTP metrics server
  - `StartMetricsServer(ctx context.Context, port string, logger *zap.Logger)` function
  - Exposes Prometheus `/metrics` endpoint
  - Respects context cancellation for graceful shutdown
  - HTTP server with configurable timeouts
  - Uses `promhttp.Handler()` for standard Prometheus format

#### Configuration Updates
- **`internal/config/config.go`**: Extended configuration
  - `QueueConfig` struct with `BufferSize int` field
  - `MetricsConfig` struct with `Port string` field
  - `Queue` and `Metrics` fields added to `Config` struct
  - `QUEUE_BUFFER_SIZE` environment variable support (default: 1000)
  - `METRICS_PORT` environment variable support (default: 8080)
  - Validation: `QUEUE_BUFFER_SIZE` must be > 0

#### Consumer Integration
- **`internal/consumer/kafka_consumer.go`**: Queue integration
  - `queue *queue.Queue` field added to `Consumer` struct
  - `NewConsumer()` signature updated to accept queue parameter
  - `Start()` method now creates `*types.Event` and enqueues to queue
  - Handles enqueue errors (context cancellation vs. other errors)
  - Logs enqueue operations with queue depth metadata
  - Backpressure automatically applied when queue is full

#### Main Application Updates
- **`cmd/main.go`**: Complete observability and queue integration
  - Metrics initialization with service name
  - Queue creation with configurable buffer size
  - Metrics HTTP server started in goroutine
  - Consumer receives queue reference
  - Improved goroutine management using `sync.WaitGroup.Go()` method
  - Graceful shutdown order: context cancellation → metrics server → queue → consumer

#### Prometheus Integration
- **`prometheus/prometheus.yml`**: Prometheus scrape configuration
  - Scrape job for `myeventstream` service
  - Target: `myeventstream:8080`
  - Metrics path: `/metrics`
  - Scrape interval: 15 seconds
  - Service and environment labels

#### Grafana Integration
- **`grafana/provisioning/datasources/prometheus.yml`**: Automatic datasource provisioning
  - Prometheus datasource configured automatically
  - URL: `http://prometheus:9090`
  - Set as default datasource
  - 15-second time interval

- **`grafana/provisioning/dashboards/dashboard.yml`**: Dashboard provisioning
  - Automatic dashboard discovery from `/var/lib/grafana/dashboards`
  - 10-second update interval
  - UI updates allowed

- **`grafana/dashboards/queue_depth.json`**: Queue depth dashboard
  - Graph panel showing queue depth over time
  - Stat panel with current queue depth and color thresholds
  - Gauge panel with visual queue depth representation
  - Thresholds: green (<500), yellow (500-800), red (>800)
  - 10-second refresh interval

#### Docker Infrastructure
- **`docker-compose.yml`**: Prometheus and Grafana services
  - Prometheus service with volume persistence
  - Grafana service with admin credentials from environment
  - Service dependencies: Grafana → Prometheus → MyEventStream
  - Port mappings: Prometheus (9090), Grafana (3000)
  - Volume mounts for configuration and data persistence
  - Network isolation within `myeventstream-network`

#### Dependencies
- Added `github.com/prometheus/client_golang v1.23.2` for Prometheus metrics

### Changed

- **`internal/consumer/kafka_consumer.go`**: 
  - Now enqueues events instead of just logging
  - Logs include queue depth information

- **`cmd/main.go`**:
  - Uses `sync.WaitGroup.Go()` for cleaner goroutine management
  - Integrated metrics and queue initialization
  - Enhanced graceful shutdown sequence

- **`.env.example`**:
  - Added `QUEUE_BUFFER_SIZE` with default value (1000)
  - Added `METRICS_PORT` with default value (8080)
  - Added `GRAFANA_ADMIN_USER` with default value (admin)
  - Added `GRAFANA_ADMIN_PASSWORD` with default value (admin)

### Technical Details

- Queue implements backpressure by blocking on `Enqueue()` when channel is full
- Metrics are updated atomically using Prometheus gauge operations
- Queue closure uses `sync.Once` to ensure idempotent behavior
- All blocking operations respect context cancellation
- Metrics server runs independently and can be scraped by Prometheus
- Grafana dashboard automatically loads on service startup
- Prometheus scrapes metrics every 15 seconds
- Queue depth metric includes service name label for multi-service scenarios

---

## [0.2.0] - Stage 0.2: Kafka Integration

### Added

#### Configuration Management
- **`internal/config/config.go`**: Configuration package with environment variable loading
  - `Config` struct with `Kafka`, `Logging`, and `Service` sub-configs
  - `Load()` function that reads from environment variables
  - Support for comma-separated Kafka broker addresses
  - Required variables: `KAFKA_BROKERS`, `KAFKA_TOPIC`
  - Optional variables with defaults: `KAFKA_GROUP_ID`, `LOG_LEVEL`, `SERVICE_NAME`

#### Structured Logging
- **`internal/logger/logger.go`**: Structured logging package using zap
  - Global `Logger` variable for application-wide logging
  - `Init(level string)` function to initialize logger with configurable log level
  - `Sync()` function for flushing buffered log entries
  - RFC3339 timestamp encoding
  - Production-ready logging configuration

#### Kafka Consumer
- **`internal/consumer/kafka_consumer.go`**: Kafka consumer implementation
  - `Consumer` struct with kafka-go reader
  - `NewConsumer()` constructor function
  - `Start(ctx context.Context)` method for consuming messages with context cancellation support
  - `Close()` method for graceful cleanup
  - Manual message handling with metadata logging
  - Respects context cancellation for graceful shutdown

#### Main Application Updates
- **`cmd/main.go`**: Complete application lifecycle management
  - Configuration loading with error handling
  - Logger initialization
  - Kafka consumer creation and management
  - Graceful shutdown with 30-second timeout
  - Signal handling for SIGINT and SIGTERM
  - Proper goroutine management for consumer

#### Docker Infrastructure
- **`docker-compose.yml`**: Kafka service integration
  - Kafka service using KRaft mode (no Zookeeper dependency)
  - Single-node Kafka setup with proper KRaft configuration
  - Network isolation with `myeventstream-network`
  - Volume persistence for Kafka data
  - Service dependencies and startup ordering
  - Environment variable support for Kafka configuration

#### Dependencies
- Added `github.com/joho/godotenv` for environment variable loading
- Added `go.uber.org/zap` for structured logging
- Added `github.com/segmentio/kafka-go` for Kafka consumer

#### Documentation
- **`docs/CONFIGURATION.md`**: Complete environment variable documentation
- **`docs/RUNNING.md`**: Local development and running instructions

### Changed

- **`.env.example`**: Updated with Kafka configuration variables
  - Added `KAFKA_BROKERS` with examples
  - Added `KAFKA_TOPIC` placeholder
  - Added `KAFKA_GROUP_ID` with default value
  - Added `LOG_LEVEL` with default value
  - Added `SERVICE_NAME` with default value
  - Added Kafka infrastructure configuration variables

- **`go.mod`**: Updated with new dependencies
  - Go version: 1.25
  - Module path: `github.com/Sheliakhin-Golang-portfolio/MyEventStream`

### Technical Details

- Consumer uses manual commit mode for full control over offset management
- All blocking operations respect context cancellation
- Metadata-only logging (topic, partition, offset, sizes, timestamp)
- Graceful error handling with retry capability
- Stateless service design (no database or persistence layer)
- Docker network security: Kafka ports exposed only within Docker network

---

## [0.1.0] - Stage 0.1: Project Initialization

### Added

#### Project Structure
- **`go.mod`**: Go module initialization
  - Module path: `github.com/Sheliakhin-Golang-portfolio/MyEventStream`
  - Go version: 1.25

#### Directory Structure
- **`cmd/`**: Application entry point directory
- **`internal/`**: Private application code directory
- **`docs/`**: Documentation directory

#### Entry Point
- **`cmd/main.go`**: Basic main function with package documentation
  - Empty `main()` function as initial placeholder
  - GoDoc package comment

#### Configuration Template
- **`.env.example`**: Environment variable template
  - Placeholder comments for Kafka broker configuration
  - Service configuration placeholders
  - Observability configuration placeholders
  - Inline documentation for each variable

#### Docker Configuration
- **`Dockerfile`**: Multi-stage build configuration
  - Builder stage using `golang:1.25-alpine`
  - Runtime stage using `alpine:latest`
  - Static binary compilation with CGO disabled
  - Binary size optimization with `-ldflags="-w -s"`
  - Healthcheck placeholder
  - Non-root user support ready

- **`docker-compose.yml`**: Initial Docker Compose setup
  - MyEventStream service definition
  - Basic healthcheck placeholder
  - Environment variable file reference
  - Port mappings for metrics endpoint

#### Version Control
- **`.gitignore`**: Git ignore patterns for Go projects

### Technical Details

- Multi-stage Docker build for optimized image size
- Proper layer caching in Dockerfile
- Minimal runtime image (Alpine Linux)
- Environment-driven configuration approach
- No hardcoded secrets or configuration values

---

## [Unreleased]

Future stages and features will be documented here as they are implemented.

### Planned Features

- Message processing pipeline (decode → validate → process)
- Retry mechanism with exponential backoff
- Dead-Letter Queue (DLQ) integration
- Additional metrics (throughput, error rates, retry rates)
- Health check implementation

---

## Notes

- All stages follow semantic versioning principles
- Each stage is designed to be runnable and maintainable
- Code follows Go 1.25+ conventions and idiomatic patterns
- Security best practices are applied throughout (network isolation, no hardcoded secrets)
- The project is stateless by design - all state is managed by the message broker
