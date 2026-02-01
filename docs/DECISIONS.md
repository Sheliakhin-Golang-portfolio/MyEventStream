# Architecture & Design Decisions

This document records **key architectural and design decisions** made during the development of MyEventStream.

The goal is not to claim a “perfect” design, but to **make trade-offs explicit**, document intent, and show how the system evolved over time.

---

## 1. Message Broker Choice

**Decision:** Use a single external broker (Kafka *or* NATS) as the source of truth for events.

**Rationale:**
- MyEventStream is not a broker replacement
- Relying on a mature broker simplifies durability and ordering concerns
- Kafka/NATS represent common, real-world infrastructure

**Alternatives Considered:**
- Supporting multiple brokers simultaneously  
  → Rejected to avoid abstraction bloat and unclear ownership of guarantees

**Implications:**
- Delivery guarantees depend on broker semantics
- MyEventStream remains stateless with respect to persistence

---

## 2. Delivery Semantics

**Decision:** At-least-once delivery.

**Rationale:**
- Matches the default behavior of most message brokers
- Significantly simpler than exactly-once semantics
- Reflects what many production systems actually use

**Non-Goals:**
- Exactly-once processing
- Deduplication at the broker level

**Implications:**
- Consumers must tolerate duplicate events
- Retry logic is explicit and visible

---

## 3. Internal Queue & Backpressure

**Decision:** Use a bounded buffered channel as the internal queue.

**Rationale:**
- Go channels provide simple, explicit backpressure
- Bounded buffers prevent unbounded memory growth
- Backpressure propagates naturally to ingestion

**Alternatives Considered:**
- Unbounded queues  
  → Rejected due to overload risk
- Custom queue implementation  
  → Rejected for unnecessary complexity

**Implications:**
- Throughput is intentionally limited by configuration
- Under load, ingestion slows instead of failing catastrophically

---

## 4. Concurrency Model

**Decision:** Fixed-size worker pool using goroutines.

**Rationale:**
- Simple and predictable concurrency
- Avoids goroutine explosion
- Easy to reason about during shutdown

**Alternatives Considered:**
- One goroutine per message  
  → Rejected due to lack of control
- Dynamic worker scaling  
  → Deferred as unnecessary early complexity

**Implications:**
- Throughput tuning is configuration-driven
- CPU and memory usage remain bounded

---

## 5. Processing Pipeline Structure

**Decision:** Explicit pipeline stages (decode → validate → process).

**Rationale:**
- Improves readability and testability
- Makes failure points obvious
- Matches how real services evolve

**Alternatives Considered:**
- Single monolithic handler  
  → Rejected for poor test isolation

**Implications:**
- Each stage can be tested independently
- Business logic can evolve without affecting ingestion

---

## 6. Retry Strategy

**Decision:** Configurable retries with exponential backoff before DLQ.

**Rationale:**
- Transient failures are common in distributed systems
- Retrying immediately is often counterproductive
- Backoff reduces pressure on downstream dependencies

**Alternatives Considered:**
- Infinite retries  
  → Rejected due to resource exhaustion risk
- No retries  
  → Rejected as unrealistic

**Implementation:**
- Exponential backoff formula: `baseDelay * (multiplier ^ attempt)`, capped at `maxDelay`
- Default: 3 retries with 100ms base, 2.0 multiplier → delays: 100ms, 200ms, 400ms
- All backoff sleeps respect `context.Context` cancellation for graceful shutdown
- Retry attempts tracked in `Event.Meta.RetryAttempt`

**Implications:**
- Message metadata tracks retry attempts
- System favors progress over perfection
- Configurable via environment variables: `RETRY_MAX_ATTEMPTS`, `RETRY_BASE_DELAY_MS`, `RETRY_MAX_DELAY_MS`, `RETRY_MULTIPLIER`

---

## 7. Dead-Letter Queue (DLQ)

**Decision:** Failed messages are published to a dedicated DLQ Kafka topic.

**Rationale:**
- Prevents poison messages from blocking the pipeline
- Allows offline inspection and replay
- Common industry practice

**Implementation:**
- DLQ producer wraps `kafka.Writer` for publishing to DLQ topic
- Kafka headers carry metadata: `original_topic`, `original_partition`, `original_offset`, `retry_attempts`, `max_retries`, `error_message`, `timestamp`
- DLQ publish has internal retry (3 attempts) to handle transient write failures
- On DLQ publish failure, offset is still committed to prevent infinite reprocessing (trade-off: message loss vs infinite loop)

**Non-Goals:**
- Automatic DLQ reprocessing
- DLQ persistence beyond broker guarantees

**Implications:**
- Operators must monitor DLQ volume via `dlq_messages_total` metric
- Failure is explicit, not hidden
- DLQ topic configurable via `DLQ_TOPIC` environment variable
- DLQ messages inspectable via `kafka-console-consumer` with header printing enabled

---

## 8. Observability First-Class Support

**Decision:** Built-in metrics and structured logging.

**Rationale:**
- High-throughput systems are impossible to debug without visibility
- Metrics reveal saturation and backpressure
- Logs alone are insufficient

**Implemented Signals:**
- Queue depth
- Throughput
- Error and retry rates

**Deferred:**
- Full distributed tracing (optional)

---

## 9. Graceful Shutdown

**Decision:** Use `context.Context` for lifecycle management.

**Rationale:**
- Standard Go practice
- Ensures in-flight events are processed
- Avoids message loss during redeployments

**Implications:**
- Shutdown is slower but safer
- Explicit waiting on workers

---

## 10. Configuration Strategy

**Decision:** Configuration via environment variables.

**Rationale:**
- Container-friendly
- Simple and explicit
- Matches 12-factor app principles

**Alternatives Considered:**
- Hot-reload configuration  
  → Deferred as non-essential

---

## 11. Testing Strategy

**Decision:** Test behavior, not implementation details.

**Approach:**
- Unit tests for pipeline stages, DLQ producer and retry logic
- Integration tests with broker
- Load testing with Vegeta

**Non-Goals:**
- Exhaustive fault injection
- Chaos testing

---

## 12. Scope Control

**Decision:** Explicitly limit scope to core responsibilities.

**Rationale:**
- Small, complete systems are more credible than large unfinished ones
- Demonstrates prioritization and engineering judgment

**Examples of Deferred Features:**
- Kubernetes-native scaling
- gRPC admin API
- Schema registry integration

---

## Final Note

These decisions reflect **intentional trade-offs**, not missing knowledge.

MyEventStream is designed to be:
- understandable
- observable
- failure-aware

Rather than feature-complete.

