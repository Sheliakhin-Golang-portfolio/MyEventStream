# Changelog

All notable changes to the MyEventStream project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.6.0] - Stage 0.6: Retry & Dead-Letter Queue Implementation

### Added

#### Retry Package
- `**internal/retry/retry.go**`: Retry logic with exponential backoff
  - `DoWithRetry(ctx, cfg, attempt, fn)` function that wraps operations with retry logic
  - Exponential backoff calculation: `baseDelay * (multiplier ^ attempt)` capped at `MaxDelay`
  - Context-aware: respects `ctx.Done()` for graceful shutdown
  - Returns `ErrMaxRetriesExceeded` when all retry attempts are exhausted
- `**internal/retry/retry_test.go**`: Comprehensive unit tests
  - Tests for success on first attempt, success after retries, exhausted retries
  - Context cancellation tests (pre-cancelled and during retry)
  - Backoff calculation tests with various multipliers and caps

#### Dead-Letter Queue Package
- `**internal/dlq/producer.go**`: Kafka-based DLQ producer
  - `Producer` struct wrapping `kafka.Writer` for DLQ topic
  - `NewProducer(cfg, logger)` constructor reading DLQ config from environment
  - `Publish(ctx, event, errorMsg)` method that:
    - Publishes failed events to DLQ Kafka topic
    - Encodes metadata into Kafka headers: `original_topic`, `original_partition`, `original_offset`, `retry_attempts`, `max_retries`, `error_message`, `timestamp`
    - Implements internal retry logic (3 attempts) for DLQ publish failures
    - Uses structured logging for all operations
  - `Close()` method for graceful shutdown
- `**internal/dlq/producer_test.go**`: Unit tests for DLQ producer
  - Validation tests for nil config/logger
  - Metadata extraction tests

#### Event Metadata
- `**internal/types/event.go**`: Extended `Event` struct
  - Added `Meta *EventMeta` field to carry Kafka and retry metadata
  - `EventMeta` struct with fields:
    - `Topic`, `Partition`, `Offset` for Kafka message metadata
    - `RetryAttempt`, `MaxRetries` for retry tracking

#### Configuration
- `**internal/config/config.go**`: Extended configuration
  - `RetryConfig` with `MaxAttempts`, `BaseDelayMs`, `MaxDelayMs`, `Multiplier`
  - `DLQConfig` with `Topic`, `Brokers` (defaults to main Kafka brokers)
  - Environment variable loading for:
    - `RETRY_MAX_ATTEMPTS` (default: 3)
    - `RETRY_BASE_DELAY_MS` (default: 100ms)
    - `RETRY_MAX_DELAY_MS` (default: 10000ms)
    - `RETRY_MULTIPLIER` (default: 2.0)
    - `DLQ_TOPIC` (default: myeventstream-dlq)
    - `DLQ_BROKERS` (optional, defaults to `KAFKA_BROKERS`)
- `**.env.example**`: Updated with retry and DLQ configuration examples

#### Metrics
- `**internal/obs/metrics.go**`: Added retry and DLQ metrics
  - `retry_attempts_total` counter: tracks retry attempts
  - `dlq_messages_total` counter: tracks messages sent to DLQ
  - `retry_exhausted_total` counter: tracks messages that exhausted all retries
  - Methods: `IncrementRetryAttempts()`, `IncrementDLQMessages()`, `IncrementRetryExhausted()`

### Changed

#### Kafka Consumer
- `**internal/consumer/kafka_consumer.go**`: Manual offset commit
  - Changed from `ReadMessage` (auto-commit) to `FetchMessage` (manual commit)
  - Added `CommitInterval: 0` to `ReaderConfig` for explicit commit control
  - Added `commitChan` channel and `commitLoop()` goroutine for async commits
  - `CommitMessage(msg)` method allows worker to signal completion
  - Attaches Kafka metadata (`Topic`, `Partition`, `Offset`) to `Event.Meta` before enqueuing
  - Offset committed only after successful processing or DLQ publish (at-least-once semantics)
  - `commitLoop()` is designed to drain `commitChan` channel before stopping the loop

#### Worker Pool
- `**internal/worker/pool.go**`: Integrated retry and DLQ
  - Extended `NewPool()` signature to accept `dlqProducer`, `retryConfig`, `commitFunc`, `metrics`
  - `processEvent()` wraps pipeline execution in `retry.DoWithRetry()`
  - Tracks retry attempts in `Event.Meta.RetryAttempt`
  - On retry exhaustion (`ErrMaxRetriesExceeded`):
    - Publishes event to DLQ with error details
    - Increments `retry_exhausted_total` and `dlq_messages_total` metrics
    - Commits offset to prevent reprocessing
  - On success: commits offset via `commitFunc`
  - Added helper functions: `getTopic()`, `getPartition()`, `getOffset()`, `getAttempt()`, `getMaxAttempts()`
  - All retry sleeps respect `ctx.Done()` for graceful shutdown

#### Main Application
- `**cmd/main.go**`: Wired retry and DLQ components
  - Creates `dlq.Producer` with DLQ config
  - Builds `retry.Config` from loaded environment variables
  - Passes DLQ producer, retry config, commit callback, and metrics to worker pool
  - Ensures proper shutdown order: DLQ producer closed after worker pool

### Technical Details

- Failed events are retried up to `RETRY_MAX_ATTEMPTS` times with exponential backoff
- Exponential backoff calculation: `RETRY_BASE_DELAY_MS * (RETRY_MULTIPLIER ^ attempt)`, capped at `RETRY_MAX_DELAY_MS`
- Default configuration: 3 retries with 100ms base delay, 2.0 multiplier → delays: 100ms, 200ms, 400ms
- All backoff sleeps respect context cancellation for graceful shutdown
- Messages that fail after all retries are published to `DLQ_TOPIC` with detailed metadata headers
- DLQ message headers include: `original_topic`, `original_partition`, `original_offset`, `retry_attempts`, `max_retries`, `error_message`, `timestamp`
- Operators must monitor `dlq_messages_total` metric for failed message tracking
- Offsets are committed only after successful processing or DLQ publish (at-least-once delivery semantics)
- DLQ publish failures are logged but do not block processing (offset committed to prevent infinite reprocessing)
- Trade-off: potential message loss vs infinite reprocessing loop when DLQ publish fails

---

## [0.5.0] - Stage 0.5: Processing Pipeline Implementation

### Added

#### Processing Pipeline Package

- `**internal/pipeline/decode.go**`: JSON decoding stage
  - `Payload` struct with `Id string` field (`json:"Id"`)
  - `Decode(ctx context.Context, value []byte) (*Payload, error)` function
  - Uses `encoding/json` to unmarshal Kafka message values into `Payload`
  - Returns typed `DecodeError` on malformed/invalid JSON
  - Respects `context.Context` cancellation
- `**internal/pipeline/validate.go**`: Payload validation stage
  - `Validate(ctx context.Context, payload *Payload) error` function
  - Ensures `Id` field is present and non-empty
  - Returns typed `ValidationError` on validation failures
  - Respects `context.Context` cancellation
- `**internal/pipeline/process.go**`: Business logic stage
  - `Process(ctx context.Context, payload *Payload, logger *zap.Logger) error` function
  - Mock business logic that logs processed events with structured logging
  - Deterministic logging (same input → same log output)
  - Returns typed `ProcessError` on processing failures
  - Respects `context.Context` cancellation
- `**internal/pipeline/errors.go**`: Typed errors
  - `DecodeError` for JSON decoding failures
  - `ValidationError` for payload validation failures
  - `ProcessError` for processing failures
  - Error types implemented as structs with `Error()` (and `Unwrap()` where applicable)

#### Unit Tests

- `**internal/pipeline/decode_test.go**`: Table-driven tests for decode stage
  - Valid JSON payloads
  - Invalid/malformed JSON
  - Empty JSON
  - Missing `id` field
  - Context cancellation during decode
- `**internal/pipeline/validate_test.go**`: Table-driven tests for validate stage
  - Valid payload with non-empty `Id`
  - Empty `Id`
  - Nil payload / missing `Id`
  - Context cancellation during validate
- `**internal/pipeline/process_test.go**`: Table-driven tests for process stage
  - Successful processing with deterministic log output
  - Context cancellation during process

#### Changed

- `**internal/worker/pool.go**`: Processing pipeline integration
  - `processEvent` updated to call pipeline stages sequentially:
    1. `pipeline.Decode(ctx, event.Value)`
    2. `pipeline.Validate(ctx, payload)`
    3. `pipeline.Process(ctx, payload, p.logger)`
  - Error handling and logging for each stage with worker ID and event metadata
  - Preserves existing graceful shutdown behavior and context handling

### Technical Details

- Processing pipeline implements explicit three-stage flow: decode → validate → process
- All stages use typed errors to distinguish decoding, validation, and processing failures
- All blocking/long-running operations respect `context.Context` for cancellation
- Business logic layer is deterministic and easily testable
- Worker pool delegates business processing to the pipeline while keeping lifecycle and shutdown semantics unchanged

---

## [0.4.0] - Stage 0.4: Worker Pool Implementation

### Added

#### Worker Pool Package

- `**internal/worker/pool.go**`: Fixed-size worker pool implementation
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

- `**internal/config/config.go**`: Worker pool configuration
  - `WorkerPoolConfig` struct with `Size int` field
  - `WorkerPool` field added to `Config` struct
  - `WORKER_POOL_SIZE` environment variable support (default: 10)
  - Validation: `WORKER_POOL_SIZE` must be > 0

#### Main Application Updates

- `**cmd/main.go**`: Worker pool integration
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

- `**docs/CONFIGURATION.md**`: Added `WORKER_POOL_SIZE` documentation
  - Default value: 10
  - Description: Number of concurrent workers
  - Usage context and tuning guidance

### Changed

- `**cmd/main.go**`:
  - Enhanced graceful shutdown to include worker pool
  - Worker pool stops before queue closure to ensure in-flight events complete
  - Improved shutdown timeout handling (30 seconds)
  - Better goroutine coordination using `sync.WaitGroup`
- `**.env.example**`:
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

- `**internal/types/event.go**`: Shared event type definition
  - `Event` struct with `Key []byte` and `Value []byte` fields
  - Used across consumer, queue, and future processing stages
  - Supports arbitrary data formats

#### Queue Package

- `**internal/queue/queue.go**`: Bounded buffered queue implementation
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

- `**internal/obs/metrics.go**`: Prometheus metrics implementation
  - `Metrics` struct with `QueueDepth` gauge metric
  - `NewMetrics(serviceName string)` constructor
  - `IncrementQueueDepth()` - increments queue depth metric
  - `DecrementQueueDepth()` - decrements queue depth metric
  - `NullifyQueueDepth()` - sets queue depth to 0
  - Metrics registered with Prometheus default registry
  - Service name label for metric identification
- `**internal/obs/http.go**`: HTTP metrics server
  - `StartMetricsServer(ctx context.Context, port string, logger *zap.Logger)` function
  - Exposes Prometheus `/metrics` endpoint
  - Respects context cancellation for graceful shutdown
  - HTTP server with configurable timeouts
  - Uses `promhttp.Handler()` for standard Prometheus format

#### Configuration Updates

- `**internal/config/config.go**`: Extended configuration
  - `QueueConfig` struct with `BufferSize int` field
  - `MetricsConfig` struct with `Port string` field
  - `Queue` and `Metrics` fields added to `Config` struct
  - `QUEUE_BUFFER_SIZE` environment variable support (default: 1000)
  - `METRICS_PORT` environment variable support (default: 8080)
  - Validation: `QUEUE_BUFFER_SIZE` must be > 0

#### Consumer Integration

- `**internal/consumer/kafka_consumer.go**`: Queue integration
  - `queue *queue.Queue` field added to `Consumer` struct
  - `NewConsumer()` signature updated to accept queue parameter
  - `Start()` method now creates `*types.Event` and enqueues to queue
  - Handles enqueue errors (context cancellation vs. other errors)
  - Logs enqueue operations with queue depth metadata
  - Backpressure automatically applied when queue is full

#### Main Application Updates

- `**cmd/main.go**`: Complete observability and queue integration
  - Metrics initialization with service name
  - Queue creation with configurable buffer size
  - Metrics HTTP server started in goroutine
  - Consumer receives queue reference
  - Improved goroutine management using `sync.WaitGroup.Go()` method
  - Graceful shutdown order: context cancellation → metrics server → queue → consumer

#### Prometheus Integration

- `**prometheus/prometheus.yml**`: Prometheus scrape configuration
  - Scrape job for `myeventstream` service
  - Target: `myeventstream:8080`
  - Metrics path: `/metrics`
  - Scrape interval: 15 seconds
  - Service and environment labels

#### Grafana Integration

- `**grafana/provisioning/datasources/prometheus.yml**`: Automatic datasource provisioning
  - Prometheus datasource configured automatically
  - URL: `http://prometheus:9090`
  - Set as default datasource
  - 15-second time interval
- `**grafana/provisioning/dashboards/dashboard.yml**`: Dashboard provisioning
  - Automatic dashboard discovery from `/var/lib/grafana/dashboards`
  - 10-second update interval
  - UI updates allowed
- `**grafana/dashboards/queue_depth.json**`: Queue depth dashboard
  - Graph panel showing queue depth over time
  - Stat panel with current queue depth and color thresholds
  - Gauge panel with visual queue depth representation
  - Thresholds: green (<500), yellow (500-800), red (>800)
  - 10-second refresh interval

#### Docker Infrastructure

- `**docker-compose.yml**`: Prometheus and Grafana services
  - Prometheus service with volume persistence
  - Grafana service with admin credentials from environment
  - Service dependencies: Grafana → Prometheus → MyEventStream
  - Port mappings: Prometheus (9090), Grafana (3000)
  - Volume mounts for configuration and data persistence
  - Network isolation within `myeventstream-network`

#### Dependencies

- Added `github.com/prometheus/client_golang v1.23.2` for Prometheus metrics

### Changed

- `**internal/consumer/kafka_consumer.go**`: 
  - Now enqueues events instead of just logging
  - Logs include queue depth information
- `**cmd/main.go**`:
  - Uses `sync.WaitGroup.Go()` for cleaner goroutine management
  - Integrated metrics and queue initialization
  - Enhanced graceful shutdown sequence
- `**.env.example**`:
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

- `**internal/config/config.go**`: Configuration package with environment variable loading
  - `Config` struct with `Kafka`, `Logging`, and `Service` sub-configs
  - `Load()` function that reads from environment variables
  - Support for comma-separated Kafka broker addresses
  - Required variables: `KAFKA_BROKERS`, `KAFKA_TOPIC`
  - Optional variables with defaults: `KAFKA_GROUP_ID`, `LOG_LEVEL`, `SERVICE_NAME`

#### Structured Logging

- `**internal/logger/logger.go**`: Structured logging package using zap
  - Global `Logger` variable for application-wide logging
  - `Init(level string)` function to initialize logger with configurable log level
  - `Sync()` function for flushing buffered log entries
  - RFC3339 timestamp encoding
  - Production-ready logging configuration

#### Kafka Consumer

- `**internal/consumer/kafka_consumer.go**`: Kafka consumer implementation
  - `Consumer` struct with kafka-go reader
  - `NewConsumer()` constructor function
  - `Start(ctx context.Context)` method for consuming messages with context cancellation support
  - `Close()` method for graceful cleanup
  - Manual message handling with metadata logging
  - Respects context cancellation for graceful shutdown

#### Main Application Updates

- `**cmd/main.go**`: Complete application lifecycle management
  - Configuration loading with error handling
  - Logger initialization
  - Kafka consumer creation and management
  - Graceful shutdown with 30-second timeout
  - Signal handling for SIGINT and SIGTERM
  - Proper goroutine management for consumer

#### Docker Infrastructure

- `**docker-compose.yml**`: Kafka service integration
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

- `**docs/CONFIGURATION.md**`: Complete environment variable documentation
- `**docs/RUNNING.md**`: Local development and running instructions

### Changed

- `**.env.example**`: Updated with Kafka configuration variables
  - Added `KAFKA_BROKERS` with examples
  - Added `KAFKA_TOPIC` placeholder
  - Added `KAFKA_GROUP_ID` with default value
  - Added `LOG_LEVEL` with default value
  - Added `SERVICE_NAME` with default value
  - Added Kafka infrastructure configuration variables
- `**go.mod**`: Updated with new dependencies
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

- `**go.mod**`: Go module initialization
  - Module path: `github.com/Sheliakhin-Golang-portfolio/MyEventStream`
  - Go version: 1.25

#### Directory Structure

- `**cmd/**`: Application entry point directory
- `**internal/**`: Private application code directory
- `**docs/**`: Documentation directory

#### Entry Point

- `**cmd/main.go**`: Basic main function with package documentation
  - Empty `main()` function as initial placeholder
  - GoDoc package comment

#### Configuration Template

- `**.env.example**`: Environment variable template
  - Placeholder comments for Kafka broker configuration
  - Service configuration placeholders
  - Observability configuration placeholders
  - Inline documentation for each variable

#### Docker Configuration

- `**Dockerfile**`: Multi-stage build configuration
  - Builder stage using `golang:1.25-alpine`
  - Runtime stage using `alpine:latest`
  - Static binary compilation with CGO disabled
  - Binary size optimization with `-ldflags="-w -s"`
  - Healthcheck placeholder
  - Non-root user support ready
- `**docker-compose.yml**`: Initial Docker Compose setup
  - MyEventStream service definition
  - Basic healthcheck placeholder
  - Environment variable file reference
  - Port mappings for metrics endpoint

#### Version Control

- `**.gitignore**`: Git ignore patterns for Go projects

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

