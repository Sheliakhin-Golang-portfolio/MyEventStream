# Producer script using kafka-console-producer (PowerShell)
# This script sends sample payloads to Kafka topic

param(
    [string]$PayloadFile = "payload.json"
)

# Get topic from environment or use default
$topic = if ($env:KAFKA_TOPIC) { $env:KAFKA_TOPIC } else { "topic1" }

# Check if payload file exists
if (-not (Test-Path $PayloadFile)) {
    Write-Host "Error: Payload file '$PayloadFile' not found" -ForegroundColor Red
    Write-Host "Usage: .\produce.ps1 [-PayloadFile <file>]"
    exit 1
}

# Check if running via Docker Compose
$kafkaStatus = docker compose ps kafka 2>$null | Select-String "Up"
if ($kafkaStatus) {
    Write-Host "Using Docker Compose Kafka container..."
    Get-Content $PayloadFile | docker compose exec -T kafka kafka-console-producer `
        --bootstrap-server localhost:9092 `
        --topic $topic
} else {
    # Try direct kafka-console-producer (requires Kafka CLI tools on host)
    $kafkaProducer = Get-Command kafka-console-producer -ErrorAction SilentlyContinue
    if ($kafkaProducer) {
        Write-Host "Using host kafka-console-producer..."
        $brokers = if ($env:KAFKA_BROKERS) { $env:KAFKA_BROKERS } else { "localhost:9092" }
        Get-Content $PayloadFile | kafka-console-producer `
            --bootstrap-server $brokers `
            --topic $topic
    } else {
        Write-Host "Error: kafka-console-producer not found on host" -ForegroundColor Red
        Write-Host "Please ensure Kafka is running via Docker Compose or install Kafka CLI tools"
        exit 1
    }
}

Write-Host "Sent payload from $PayloadFile to topic $topic" -ForegroundColor Green
