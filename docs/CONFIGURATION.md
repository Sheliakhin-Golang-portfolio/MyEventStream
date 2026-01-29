# Configuration & Environment Variables

This document describes all environment variables used by the MyEventStream service.

All variables listed here are preserved from the original README and configuration files.

---

## General

| Variable | Description |
|----------|-------------|
| `LOG_LEVEL` | Logging level used by the service (debug, info, warn, error) |
| `SERVICE_NAME` | Service name for logging and metrics |

---

## Message Broker (Kafka)

| Variable | Description |
|----------|-------------|
| `KAFKA_BROKERS` | Kafka broker addresses (comma-separated). Example: `localhost:9092,localhost:9093` |
| `KAFKA_TOPIC` | Kafka topic to consume from |
| `KAFKA_GROUP_ID` | Kafka consumer group ID |

Used by:
- MyEventStream service

---

## Kafka Infrastructure (Docker Compose)

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_IMAGE_VERSION` | Kafka Docker image version tag | `latest` |
| `KAFKA_AUTO_CREATE_TOPICS_ENABLE` | Automatically create topics if they don't exist | `true` |
| `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR` | Replication factor for offsets topic | `1` |
| `KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR` | Replication factor for transaction state log | `1` |
| `KAFKA_TRANSACTION_STATE_LOG_MIN_ISR` | Minimum in-sync replicas for transaction state log | `1` |

Used by:
- Kafka container (docker-compose)

**Note:** These settings are for single-node Kafka setup. For production multi-node clusters, adjust replication factors accordingly.

---

## Grafana Configuration (Docker Compose)

| Variable | Description | Default |
|----------|-------------|---------|
| `GRAFANA_ADMIN_USER` | Grafana admin username | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | `admin` |

Used by:
- Grafana container (docker-compose)

**Note:** Change the default password in production. These map to `GF_SECURITY_ADMIN_USER` and `GF_SECURITY_ADMIN_PASSWORD` in the Grafana container.

---

## Service Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_PORT` | Metrics server port | `8080` |
| `WORKER_POOL_SIZE` | Number of concurrent workers | `10` |
| `QUEUE_BUFFER_SIZE` | Internal queue buffer size (backpressure control) | `1000` |

Used by:
- MyEventStream service

---

## Retry Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `RETRY_MAX_ATTEMPTS` | Maximum number of retry attempts before sending to DLQ | `3` |
| `RETRY_BASE_DELAY_MS` | Base delay in milliseconds for exponential backoff | `100` |
| `RETRY_MAX_DELAY_MS` | Maximum delay in milliseconds (cap for exponential backoff) | `10000` |
| `RETRY_MULTIPLIER` | Exponential backoff multiplier | `2.0` |

Used by:
- MyEventStream service

**Retry Behavior:**
- Failed events are retried with exponential backoff: `RETRY_BASE_DELAY_MS * (RETRY_MULTIPLIER ^ attempt)`
- Backoff is capped at `RETRY_MAX_DELAY_MS`
- Default configuration: 3 retries with delays of 100ms, 200ms, 400ms
- All retry operations respect context cancellation for graceful shutdown

---

## Dead-Letter Queue Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `DLQ_TOPIC` | Kafka topic for failed messages (Dead-Letter Queue) | `myeventstream-dlq` |
| `DLQ_BROKERS` | Kafka broker addresses for DLQ (optional, defaults to `KAFKA_BROKERS`) | — |

Used by:
- MyEventStream service

**DLQ Behavior:**
- Messages that exhaust all retry attempts are published to the DLQ topic
- Each DLQ message includes metadata headers: `original_topic`, `original_partition`, `original_offset`, `retry_attempts`, `max_retries`, `error_message`, `timestamp`
- DLQ publishing has internal retry logic (3 attempts) to ensure delivery
- If DLQ publish fails after retries, the message is lost but offset is committed to prevent infinite reprocessing

---

## Notes for Reviewers

- Environment variables are loaded at startup
- No configuration is hardcoded
- Secrets are never committed to the repository
- Copy `.env.example` to `.env` and update with your actual values
