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

### Verifying that messages are processed

In Kafka, **consuming does not delete messages** from the topic. The log is append-only; messages remain until retention (time or size). So the **total message count** of a topic does **not** decrease when MyEventStream consumes.

To verify that the service is processing messages:

- **Consumer group lag** — Should go to 0 (or stay low) when the consumer keeps up:
  ```bash
  docker compose exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group myeventstream-consumer
  ```
- **Application metrics** — `events_ingested_total` and `events_processed_total` at `http://localhost:8080/metrics` (or your `METRICS_PORT`) increase as messages are consumed and processed.
- **Application logs** — Look for "Enqueued message" and "Event processed successfully" in the myeventstream container logs.

---

<a id="observability"></a>
## Observability (Prometheus & Grafana)

When running via Docker Compose, Prometheus and Grafana are available for monitoring.

| Service    | URL                     | Purpose                                   |
|------------|-------------------------|-------------------------------------------|
| Prometheus | http://localhost:9090   | Query metrics, targets, and scrape config |
| Grafana    | http://localhost:3000   | Dashboards and visualization              |

### Metrics exposed at `/metrics`

The application exposes the following Prometheus metrics (all include a `service` label):

| Metric                    | Type    | Description                                                                 |
|---------------------------|---------|-----------------------------------------------------------------------------|
| `queue_depth`             | Gauge   | Current depth of the internal event queue (backpressure visibility)         |
| `events_ingested_total`   | Counter | Total number of events ingested from the broker into the internal queue     |
| `events_processed_total`  | Counter | Total number of events successfully processed by the pipeline               |
| `retry_attempts_total`    | Counter | Total number of retry attempts for failed events                            |
| `dlq_messages_total`      | Counter | Total number of messages sent to the dead-letter queue                      |
| `retry_exhausted_total`   | Counter | Total number of messages that exhausted all retry attempts                  |

**Grafana** is pre-provisioned with:

- A **Prometheus** datasource (points to `http://prometheus:9090`)
- The **MyEventStream Queue Depth** dashboard, which visualizes the `queue_depth` metric

All metrics above are available in Prometheus for ad-hoc queries and custom dashboards. Log in with the credentials from your `.env`: `GRAFANA_ADMIN_USER` and `GRAFANA_ADMIN_PASSWORD` (defaults: `admin` / `admin`). See [CONFIGURATION.md](CONFIGURATION.md) for details.

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
- Metrics (queue depth, ingestion/processing counts, retries, DLQ) are exposed at `/metrics` and scraped by Prometheus when using Docker Compose; see [Observability](#observability) for the full list

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

## Inspecting Dead-Letter Queue Messages

Messages that fail after all retry attempts are sent to the Dead-Letter Queue (DLQ) for manual inspection.

To inspect DLQ messages, use `kafka-console-consumer`:

```bash
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic myeventstream-dlq \
  --from-beginning \
  --property print.headers=true \
  --property print.timestamp=true
```

**When running via Docker Compose:**

If Kafka is running in Docker Compose (and port 9092 is not published to the host), you can access `kafka-console-consumer` through the Kafka container:

```bash
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic myeventstream-dlq \
  --from-beginning \
  --property print.headers=true \
  --property print.timestamp=true
```

The Confluent Kafka image includes all Kafka CLI tools, so `kafka-console-consumer` is available inside the container.

Each DLQ message includes the following headers:

- `original_topic` — the original Kafka topic
- `original_partition` — the original partition number
- `original_offset` — the original offset
- `retry_attempts` — number of retry attempts made
- `max_retries` — maximum retries configured
- `error_message` — the final error that caused the message to be sent to DLQ
- `timestamp` — when the message was published to DLQ

These headers provide full traceability for debugging failed messages.

---

## Notes for Reviewers

- The project is designed to be run locally without external dependencies
- Docker is used for infrastructure (Kafka, Prometheus, Grafana) and optionally for the application service
- Configuration is explicit and environment-driven
- The service processes events with controlled concurrency and backpressure; metrics (queue depth, ingestion/processing, retries, DLQ) are exposed via Prometheus and the queue depth dashboard is available in Grafana
- All failed messages are captured in the Dead-Letter Queue for manual inspection
