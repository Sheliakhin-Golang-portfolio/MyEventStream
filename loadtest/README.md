# Load Testing Guide

This directory contains tools and documentation for load testing MyEventStream.

## Prerequisites

- **Docker** and **Docker Compose** (for running Kafka and infrastructure)
- **Go 1.25+** (for running the Go producer)
- **Vegeta** (optional, for HTTP load testing): `go install github.com/tsenart/vegeta/v12@latest`

## Docker Compose Setup for Load Testing

By default, `docker-compose.yml` does **not** publish Kafka port 9092 to the host (containers talk via `kafka:9092`). To run load test tools (Go producer, scripts, Vegeta) **on the host** against Kafka, you must enable host access:

1. **In `MyEventStream/docker-compose.yml`**, under the `kafka` service:
   - **KAFKA_ADVERTISED_LISTENERS**: Comment out the default and use the host listener for tests.
     - Comment out: `KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092`
     - Uncomment: `KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://host.docker.internal:9092`
   - **ports**: Uncomment so port 9092 is published to the host:
     ```yaml
     ports:
       - "9092:9092"
     ```
   - **extra_hosts**: Uncomment so Kafka can resolve `host.docker.internal`:
     ```yaml
     extra_hosts:
       - "host.docker.internal:host-gateway"
     ```

2. Restart the stack: `docker compose down && docker compose up -d`.

After tests, you can revert these changes (comment ports and extra_hosts, and set `KAFKA_ADVERTISED_LISTENERS` back to `PLAINTEXT://kafka:9092`) so that Kafka is not exposed to the host in normal operation.

## Overview

Load testing can be performed using multiple methods:

1. **kafka-console-producer scripts** (`produce.sh` / `produce.ps1`) - Simple script-based approach
2. **Go producer CLI** - Programmatic load generation with rate limiting and batch control
3. **Go producer HTTP mode** - HTTP endpoint for Vegeta-based load testing

All methods produce messages to the Kafka topic configured in `KAFKA_TOPIC` environment variable.

---

## Producing Load into Kafka

### Method 1: Using kafka-console-producer Scripts

#### Bash Script (`produce.sh`)

```bash
# Set environment variables (or use .env file)
export KAFKA_TOPIC=topic1

# Run from loadtest directory
cd MyEventStream/loadtest
chmod +x produce.sh
./produce.sh payload.json

# Or specify a different payload file
./produce.sh payload-2.json
```

The script automatically detects if Kafka is running via Docker Compose and uses `docker compose exec` accordingly.

#### PowerShell Script (`produce.ps1`)

```powershell
# Set environment variables
$env:KAFKA_TOPIC = "topic1"

# Run from loadtest directory
cd MyEventStream\loadtest
.\produce.ps1 -PayloadFile payload.json

# Or use default payload.json
.\produce.ps1
```

**Note:** By default, Kafka port 9092 is not published to the host. For scripts running on the host, follow [Docker Compose Setup for Load Testing](#docker-compose-setup-for-load-testing) to uncomment ports and extra_hosts and set `KAFKA_ADVERTISED_LISTENERS`, then use `KAFKA_BROKERS=localhost:9092`. Otherwise, the scripts use `docker compose exec` to access Kafka inside the container.

---

### Method 2: Using Go Producer CLI

The Go producer provides flexible load generation with rate limiting and batch control. It runs **on the host** (not inside a container). For it to reach Kafka, enable host access as described in [Docker Compose Setup for Load Testing](#docker-compose-setup-for-load-testing) (uncomment ports, extra_hosts, and set `KAFKA_ADVERTISED_LISTENERS`), then use `KAFKA_BROKERS=localhost:9092`.

#### Basic Usage

```bash
cd MyEventStream/loadtest

# Set environment variables (producer runs on host; use localhost when Kafka port is published)
export KAFKA_BROKERS=localhost:9092
export KAFKA_TOPIC=topic1

# Produce messages as fast as possible (infinite)
go run ./cmd/producer

# Produce 1000 messages
go run ./cmd/producer -batch 1000

# Produce at 100 messages/second for 30 seconds
go run ./cmd/producer -rate 100 -duration 30s

# Use specific payload files
go run ./cmd/producer payload.json payload-2.json
```

#### Command-Line Options

- `-brokers`: Kafka broker addresses (comma-separated). Default: `KAFKA_BROKERS` env var or `localhost:9092` (use 9092 when Kafka runs in Docker and producer runs on host)
- `-topic`: Kafka topic name. Default: `KAFKA_TOPIC` env var or `topic1`
- `-batch`: Number of messages to produce (0 = infinite). Default: 0
- `-rate`: Messages per second (0 = as fast as possible). Default: 0
- `-duration`: Duration to run (e.g., `30s`, `5m`). If set, overrides batch limit. Default: 0
- `-http`: HTTP server port (e.g., `:8081`). If set, starts HTTP producer mode instead of CLI mode

#### Examples

```bash
# Produce 5000 messages at 200 msg/s
go run ./cmd/producer -batch 5000 -rate 200

# Run for 2 minutes at 50 msg/s
go run ./cmd/producer -rate 50 -duration 2m

# Produce 10000 messages as fast as possible
go run ./cmd/producer -batch 10000
```

---

### Method 3: HTTP Producer for Vegeta

The Go producer can run in HTTP mode to accept POST requests and produce them to Kafka. This enables Vegeta-based load testing.

#### Starting HTTP Producer

```bash
cd MyEventStream/loadtest

# Set environment variables
export KAFKA_BROKERS=localhost:9092
export KAFKA_TOPIC=topic1

# Start HTTP producer on port 8081
go run ./cmd/producer -http :8081
```

The HTTP server accepts POST requests at:
- `POST http://localhost:8081/`

Request body should be JSON matching the pipeline format: `{"Id":"..."}`

#### Running Vegeta Attack

```bash
# Install Vegeta (if not already installed)
go install github.com/tsenart/vegeta/v12@latest

# Create targets file (already provided: vegeta-targets.txt)
# Contains:
# POST http://localhost:8081/

# Run attack: 100 requests/second for 30 seconds
echo "POST http://localhost:8081/" | vegeta attack -rate=100 -duration=30s > results.bin

# View results
vegeta report results.bin

# Plot results (requires gnuplot)
vegeta plot results.bin > plot.html
```

#### Vegeta Examples

```bash
# High rate: 1000 req/s for 10 seconds
echo "POST http://localhost:8081/" | vegeta attack -rate=1000 -duration=10s -body=@payload.json > results.bin

# Sustained load: 50 req/s for 5 minutes
echo "POST http://localhost:8081/" | vegeta attack -rate=50 -duration=5m -body=@payload.json > results.bin

# With custom payload per request (using vegeta targets file)
vegeta attack -rate=100 -duration=30s -targets=vegeta-targets.txt > results.bin
```

**Note:** When using `-body=@payload.json`, Vegeta sends the same payload for all requests. For dynamic payloads, use a custom Vegeta target generator or modify the HTTP producer to generate payloads.

#### Vegeta on Windows

On Windows, shell redirection and file references work differently:

- **Output:** Replace `> results.bin` with the `-output` flag. Example:
  ```powershell
  echo "POST http://localhost:8081/" | vegeta attack -rate=100 -duration=30s -output results.bin
  ```
- **Targets and body:** Use full paths for `-targets` and `-body` instead of relative file names and `@` syntax:
  - Use `-targets=C:\path\to\vegeta-targets.txt` (full path) instead of `-targets=vegeta-targets.txt`
  - Use `-body=C:\path\to\payload.json` (full path) instead of `-body=@payload.json`

Example (PowerShell, from loadtest directory):
```powershell
$loadtestDir = (Get-Location).Path
echo "POST http://localhost:8081/" | vegeta attack -rate=100 -duration=30s -body="$loadtestDir\payload.json" -output "$loadtestDir\results.bin"
vegeta report "$loadtestDir\results.bin"
```

---

## Recording Throughput vs Workers

This section documents how to measure consumer throughput as `WORKER_POOL_SIZE` varies.

### Step-by-Step Procedure

1. **Start Infrastructure**

   If the producer will run on the host (e.g. Go producer or scripts), apply [Docker Compose Setup for Load Testing](#docker-compose-setup-for-load-testing) first (uncomment ports, extra_hosts, and set `KAFKA_ADVERTISED_LISTENERS` in `docker-compose.yml`).

   ```bash
   cd MyEventStream
   docker compose up -d
   ```

   This starts Kafka, MyEventStream, Prometheus, and Grafana.

2. **Configure Worker Pool Size**

   Edit `.env` file or set environment variable:

   ```bash
   export WORKER_POOL_SIZE=10
   ```

   Or modify `docker-compose.yml` to pass the environment variable.

3. **Start Producer**

   Choose one of the methods above. For consistent testing, use the Go producer with a fixed rate:

   ```bash
   cd loadtest
   go run ./cmd/producer -rate 100 -duration 60s
   ```

4. **Query Prometheus for Consumer Throughput**

   Prometheus is available at `http://localhost:9090`.

   Query for processing rate:

   ```
   rate(events_processed_total[1m])
   ```

   Or total processed in a time window:

   ```
   increase(events_processed_total[1m])
   ```

   **Via Prometheus UI:**
   - Navigate to `http://localhost:9090`
   - Enter query: `rate(events_processed_total[1m])`
   - View graph or table

   **Via curl:**

   ```bash
   curl 'http://localhost:9090/api/v1/query?query=rate(events_processed_total[1m])'
   ```

5. **Repeat for Different Worker Pool Sizes**

   - Stop the producer and MyEventStream service
   - Change `WORKER_POOL_SIZE` (e.g., 2, 5, 10, 20, 50)
   - Restart MyEventStream: `docker compose restart myeventstream`
   - Run the same producer load again
   - Record the throughput metric

6. **Record Results**

   Document findings in the Test Results section below.

### Key Metrics to Monitor

- **`events_processed_total`**: Total events successfully processed (counter)
- **`events_ingested_total`**: Total events ingested into queue (counter)
- **`queue_depth`**: Current queue depth (gauge) - indicates backpressure
- **`retry_attempts_total`**: Total retry attempts (counter)
- **`dlq_messages_total`**: Messages sent to dead-letter queue (counter)

Use `rate()` for throughput (events/second) and `increase()` for total counts over a window.

---

## Test Results

### Throughput vs Workers

| Worker Pool Size | Producer Rate | Consumer Throughput (events/s) | Queue Depth (max) | Average CPU usage (Docker %) | Memory usage (avg) | Notes |
|-----------------|---------------|------------------------------|-------------------|-------|
| 2               | 500 msg/s     | 441.18                        | 228             | 13.58 | 9.59 Mb | Baseline |
| 5               | 500 msg/s     | 493.33                        | 204             | 14.67 | 12.53 Mb | |
| 10              | 500 msg/s     | 500                        | 160             | 15.77 | 22.8 Mb | Default |
| 20              | 500 msg/s     | 500                        | 117             | 14.38 | 25.4 Mb | |
| 50              | 500 msg/s     | 500                       | 114             | 17 | 28.88 Mb | |

### Bottlenecks and Observations

**Queue Saturation and Stability:**
- Queue depth stabilizes and decreases as worker count increases beyond 10 workers. At 2 workers, max queue depth reached 228, but with 10+ workers, queue depth decreases significantly (160 → 117 → 114), indicating the system efficiently processes messages without accumulating backlog.
- Even though max Queue Depth has been caught for each test, the depth larger than 0 only occurred once or twice during the test, meaning that in overall the MyEventStream copes with the load effectively. The system maintains stable operation with minimal queue buildup.

**Backpressure and Queue Management:**
- With queue size set to 1000, in every test the program has managed to free the queue before it became full. No producer blocking was observed, indicating that the consumer throughput matches or exceeds the producer rate once sufficient workers (≥10) are allocated.
- At 2 workers, throughput (441.18 events/s) was below producer rate (500 msg/s), causing queue accumulation. However, with 5+ workers, throughput approaches or matches the producer rate, preventing queue saturation.

**CPU and Memory Resource Usage:**
- CPU usage remains relatively stable across all worker configurations (13.58% - 17%), showing efficient resource utilization without significant overhead from additional workers.
- Memory usage increases linearly with worker pool size (9.59 MB → 28.88 MB), which is expected behavior. The memory footprint remains reasonable even at 50 workers.

**Retry Rate:**
- There were no retries observed during any of the test runs, indicating that event processing is reliable and consistent under the tested load conditions.

**DLQ Rate:**
- There were no DLQ (Dead Letter Queue) messages recorded during testing, confirming that all events were successfully processed without permanent failures.

---

## Troubleshooting

### Kafka Connection Issues

If the Go producer cannot connect to Kafka:

1. **Check Kafka is running:**
   ```bash
   docker compose ps kafka
   ```

2. **Verify broker address:** The Go producer runs on the host. Use `KAFKA_BROKERS=localhost:9092` (or `-brokers localhost:9092`). For host access to work, ensure you have applied the [Docker Compose Setup for Load Testing](#docker-compose-setup-for-load-testing) (ports and extra_hosts uncommented, `KAFKA_ADVERTISED_LISTENERS` set to `PLAINTEXT://host.docker.internal:9092`).

3. **Check topic exists:**
   ```bash
   docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
   ```

### HTTP Producer Not Responding

- Ensure HTTP producer is running: `go run ./cmd/producer -http :8081`
- Check port is not in use: `netstat -an | grep 8081` (Linux) or `netstat -an | findstr 8081` (Windows)
- Verify Vegeta targets file points to correct URL

### Prometheus Metrics Not Appearing

- Check MyEventStream is running: `docker compose ps myeventstream`
- Verify metrics endpoint: `curl http://localhost:8080/metrics`
- Check Prometheus targets: `http://localhost:9090/targets`
- Ensure scrape interval allows time for metrics to appear (default: 15s)

---

## Additional Notes

- **Sample Payloads**: The `payload.json`, `payload-2.json`, and `payload-3.json` files contain valid JSON matching the pipeline format: `{"Id":"..."}`
- **Graceful Shutdown**: All producers support graceful shutdown via SIGTERM/SIGINT (Ctrl+C)
- **Rate Limiting**: The Go producer uses a ticker for rate limiting, which may have slight jitter under high load
