# Running the Project Locally

This document explains how to run the MyEventStream project locally for development.

It preserves the setup and execution information from the original README, reorganized for clarity and ease of use.

---

## Prerequisites

Before running the project, ensure you have:

- Go (version 1.25 or compatible)
- Docker
- Docker Compose

These tools are required to run infrastructure dependencies and services locally.

---

## Project Dependencies

The application relies on the following external services:

- **Kafka** — message broker for event consumption

These dependencies are expected to run before starting the application service.

---

## Configuration

All configuration is provided via **environment variables**.

Environment variables are documented in detail here:

➡️ **[CONFIGURATION.md](CONFIGURATION.md)**

No secrets or configuration values are hardcoded in the codebase.

Before running the service, copy `.env.example` to `.env` and update with your actual values:

```bash
cp .env.example .env
```

---

## Starting Infrastructure Services

Infrastructure services can be started using Docker Compose.

From the MyEventStream directory:

```bash
docker compose up -d --build
```

This starts:
- Kafka (KRaft mode, single-node setup)
- MyEventStream service

After that the application is ready to use.

---

## Running Application Service Separately

You can also start the service independently. For that, from the MyEventStream directory, run:

```bash
go run ./cmd
```

The service:
- reads configuration from environment variables
- exposes a metrics endpoint on the configured port (default: 8080)
- can be restarted independently

Before running the service independently, make sure that Kafka is already running. You can start only Kafka using:

```bash
docker compose up -d kafka
```

---

## Development Notes

- The service is intentionally stateless and can be restarted without data loss
- Logs are written to stdout for local development
- Errors are returned with structured messages
- Failed messages are retried automatically according to configuration
- Messages that fail after all retries are sent to the Dead-Letter Queue (DLQ)
- The service implements graceful shutdown with a 30-second timeout

---

## Stopping the Project

To stop infrastructure services:

```bash
docker compose down
```

To stop only the application service while keeping Kafka running:

```bash
docker compose stop myeventstream
```

The application service can be stopped using standard process termination (Ctrl+C) when running independently.

---

## Notes for Reviewers

- The project is designed to be run locally without external dependencies
- Docker is used for infrastructure (Kafka) and optionally for the application service
- Configuration is explicit and environment-driven
- The service processes events with controlled concurrency and backpressure
- All failed messages are captured in the Dead-Letter Queue for manual inspection
