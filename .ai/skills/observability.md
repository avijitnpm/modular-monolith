# Observability Patterns

## OpenTelemetry

Use:
- OpenTelemetry Go SDK
- OTLP exporters
- chi middleware instrumentation
- pgx instrumentation

---

## Tracing Rules

Every request should:
- generate trace context
- propagate trace IDs
- create request spans

Important spans:
- HTTP request spans
- DB query spans
- middleware spans

---

## Logging

Structured logs should include:
- trace_id
- span_id
- request path
- request method

---

## OpenObserve

Tracing should export using:
- OTLP HTTP exporter

Environment-driven configuration only.

---

## Middleware Rules

Observability middleware should:
- remain lightweight
- avoid business logic
- only enrich tracing/logging context

---

## Failure Rules

Tracing failures should:
- never crash application
- degrade gracefully