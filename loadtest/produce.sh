#!/bin/bash
# Producer script using kafka-console-producer
# This script sends sample payloads to Kafka topic

set -e

# Get topic from environment or use default
TOPIC="${KAFKA_TOPIC:-topic1}"
PAYLOAD_FILE="${1:-payload.json}"

# Check if payload file exists
if [ ! -f "$PAYLOAD_FILE" ]; then
    echo "Error: Payload file '$PAYLOAD_FILE' not found"
    echo "Usage: $0 [payload-file]"
    exit 1
fi

# Check if running via Docker Compose
if docker compose ps kafka 2>/dev/null | grep -q "Up"; then
    echo "Using Docker Compose Kafka container..."
    docker compose exec -T kafka kafka-console-producer \
        --bootstrap-server localhost:9092 \
        --topic "$TOPIC" < "$PAYLOAD_FILE"
else
    # Try direct kafka-console-producer (requires Kafka CLI tools on host)
    if command -v kafka-console-producer &> /dev/null; then
        echo "Using host kafka-console-producer..."
        BROKERS="${KAFKA_BROKERS:-localhost:9092}"
        kafka-console-producer \
            --bootstrap-server "$BROKERS" \
            --topic "$TOPIC" < "$PAYLOAD_FILE"
    else
        echo "Error: kafka-console-producer not found on host"
        echo "Please ensure Kafka is running via Docker Compose or install Kafka CLI tools"
        exit 1
    fi
fi

echo "Sent payload from $PAYLOAD_FILE to topic $TOPIC"
