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
- **Prometheus** — metrics collection (scrapes the MyEventStream `/metrics` endpoint)
- **Grafana** — dashboards and visualization (uses Prometheus as datasource)

Kafka is required before starting the application service. Prometheus and Grafana are optional observability components started via Docker Compose.

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

- **Kafka** — KRaft mode, single-node setup (internal ports 9092, 9093)
- **MyEventStream** — application service; metrics on port **8080** (configurable via `METRICS_PORT`)
- **Prometheus** — metrics scraper; UI on **http://localhost:9090**
- **Grafana** — dashboards; UI on **http://localhost:3000**

After that the application is ready to use.

---

## Observability (Prometheus & Grafana)

When running via Docker Compose, Prometheus and Grafana are available for monitoring.

| Service    | URL                     | Purpose                                   |
|------------|-------------------------|-------------------------------------------|
| Prometheus | http://localhost:9090   | Query metrics, targets, and scrape config |
| Grafana    | http://localhost:3000   | Dashboards and visualization              |

**Grafana** is pre-provisioned with:

- A **Prometheus** datasource (points to `http://prometheus:9090`)
- The **MyEventStream Queue Depth** dashboard, which visualizes the `queue_depth` metric

Log in with the credentials from your `.env`: `GRAFANA_ADMIN_USER` and `GRAFANA_ADMIN_PASSWORD` (defaults: `admin` / `admin`). See [CONFIGURATION.md](CONFIGURATION.md) for details.

---

## Running Application Service Separately

You can also start the service independently. For that, from the MyEventStream directory, run:

```bash
go run ./cmd
```

The service:

- reads configuration from environment variables
- exposes a **`/metrics`** endpoint on the configured port (default: 8080) for Prometheus scraping
- can be restarted independently

Before running the service independently, ensure **Kafka is already running** and reachable at the addresses in `KAFKA_BROKERS`. To start only Kafka (and optionally Prometheus/Grafana):

```bash
docker compose up -d kafka
```

If Kafka runs in Docker, ensure port 9092 is published to the host (e.g. add `ports: ["9092:9092"]` to the Kafka service in `docker-compose.yml`) so that `go run ./cmd` can connect.

---

## Development Notes

- The service is intentionally stateless and can be restarted without data loss
- Logs are written to stdout for local development
- Errors are returned with structured messages
- Failed messages are retried automatically according to configuration
- Messages that fail after all retries are sent to the Dead-Letter Queue (DLQ)
- The service implements graceful shutdown with a 30-second timeout
- Concurrency is controlled via `WORKER_POOL_SIZE` (default: 10); adjust this to tune throughput based on your workload
- Metrics (including `queue_depth` for backpressure visibility) are exposed at `/metrics` and scraped by Prometheus when using Docker Compose

---

## Stopping the Project

To stop all services (Kafka, MyEventStream, Prometheus, Grafana):

```bash
docker compose down
```

To stop only the application service while keeping Kafka (and optionally Prometheus/Grafana) running:

```bash
docker compose stop myeventstream
```

To stop only Prometheus and Grafana:

```bash
docker compose stop prometheus grafana
```

The application service can be stopped using standard process termination (Ctrl+C) when running independently.

---

## Notes for Reviewers

- The project is designed to be run locally without external dependencies
- Docker is used for infrastructure (Kafka, Prometheus, Grafana) and optionally for the application service
- Configuration is explicit and environment-driven
- The service processes events with controlled concurrency and backpressure; queue depth is exposed via Prometheus and visualized in Grafana
- All failed messages are captured in the Dead-Letter Queue for manual inspection
