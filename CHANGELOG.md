# Changelog

All notable changes to the MyEventStream project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

- Worker pool implementation for controlled concurrency
- Internal buffered queue for backpressure control
- Message processing pipeline (decode → validate → process)
- Retry mechanism with exponential backoff
- Dead-Letter Queue (DLQ) integration
- Metrics and observability endpoints
- Health check implementation

---

## Notes

- All stages follow semantic versioning principles
- Each stage is designed to be runnable and maintainable
- Code follows Go 1.25+ conventions and idiomatic patterns
- Security best practices are applied throughout (network isolation, no hardcoded secrets)
- The project is stateless by design - all state is managed by the message broker
