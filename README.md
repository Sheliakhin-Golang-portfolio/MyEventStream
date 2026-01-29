# MyEventStream

A production-style event processing service written in Go that consumes events from a message broker, applies controlled concurrency and backpressure, and processes messages reliably with retries and a dead-letter queue.

---

## What This Project Demonstrates

This project was created as a portfolio application with a focus on backend engineering practices for event-driven systems.

It demonstrates:

- Building stateless event processing services in Go
- Controlled concurrency and backpressure mechanisms
- Reliable message processing with retry logic and dead-letter queues
- At-least-once delivery semantics
- Observability as a first-class concern (metrics, logging)
- Graceful shutdown and lifecycle management
- Production-oriented error handling and failure recovery

---

## Product Overview

MyEventStream is an event processing service designed to consume events from message brokers (Kafka or NATS), process them through a controlled pipeline, and handle failures gracefully.

The service architecture follows a clear flow:

- **Ingestion Layer** consumes events from the broker
- **Internal Buffered Queue** provides backpressure control
- **Worker Pool** processes events with controlled concurrency
- **Processing Pipeline** applies decode → validate → process stages
- **Retry Logic** handles transient failures with backoff
- **Dead-Letter Queue** captures permanently failed messages

The project is designed to demonstrate how real Go backend services are built and evolved, not to replace Kafka or NATS. EventStream focuses on at-least-once delivery, controlled concurrency, and operational visibility — the core problems most Go services solve in production.

**Current Stage:** Stage 0.6 - Retry & Dead-Letter Queue

---

## Architecture Overview

At a high level:

- **Message Broker** (Kafka/NATS) acts as the source of truth for events
- **Ingestion Layer** consumes events with controlled rate limiting
- **Bounded Internal Queue** provides backpressure to prevent unbounded memory growth
- **Worker Pool** processes events with fixed-size concurrency
- **Processing Pipeline** applies explicit stages (decode, validate, process)
- **Retry Mechanism** handles transient failures with exponential backoff
- **Dead-Letter Queue** captures messages that fail after all retries

The service is **stateless** — no internal database or persistence layer. All state is managed by the message broker.

Detailed architectural decisions are documented separately.

---

## Documentation

All detailed documentation is moved to `docs/` folder:

- [Configuration & Environment Variables](docs/CONFIGURATION.md)
- [Running Locally](docs/RUNNING.md)

---

## CHANGELOG

As there are no multiple releases during development, all version changes are contained in [CHANGELOG.md](CHANGELOG.md) file.

---

## Project Status

This project is developed for portfolio and learning purposes.  
The focus is correctness, clarity, and realistic backend patterns.  
**NOT FOR COMMERCIAL USE!**