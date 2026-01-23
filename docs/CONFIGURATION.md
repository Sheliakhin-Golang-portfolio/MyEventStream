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

## Service Configuration

| Variable | Description |
|----------|-------------|
| `METRICS_PORT` | Metrics server port |
| `WORKER_POOL_SIZE` | Number of concurrent workers |
| `QUEUE_BUFFER_SIZE` | Internal queue buffer size (backpressure control) |

Used by:
- MyEventStream service

---

## Retry Configuration

| Variable | Description |
|----------|-------------|
| `MAX_RETRY_ATTEMPTS` | Maximum number of retry attempts before DLQ |
| `RETRY_BACKOFF_BASE` | Retry backoff base duration in seconds |

Used by:
- MyEventStream service

---

## Dead-Letter Queue Configuration

| Variable | Description |
|----------|-------------|
| `DLQ_TOPIC` | Dead-Letter Queue topic name |

Used by:
- MyEventStream service

---

## Notes for Reviewers

- Environment variables are loaded at startup
- No configuration is hardcoded
- Secrets are never committed to the repository
- Copy `.env.example` to `.env` and update with your actual values
