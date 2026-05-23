# Modular Monolith AI Agent Guide

## Architecture

This project is a modular monolith built with:

- Go 1.22+
- Chi Router
- PostgreSQL 17
- pgx
- Tern migrations
- SvelteKit static frontend
- Docker-based infrastructure

The architecture follows:

- middleware-driven request lifecycle
- service/repository separation
- interface-first external providers
- transaction-safe workflows
- multi-tenant SaaS design
- explicit dependency direction

---

# Request Lifecycle

HTTP Request
↓
Chi Router
↓
Middleware
↓
Auth Context
↓
Tenant Context
↓
Handler
↓
Validation
↓
Service Layer
↓
Repository Layer
↓
PostgreSQL

---

# Rules

## Repositories

Repositories:
- only execute persistence logic
- do not contain business logic
- do not depend on services
- do not depend on handlers

## Services

Services:
- orchestrate workflows
- own transaction boundaries
- coordinate repositories

## Middleware

Middleware:
- enriches request context
- handles auth/tenant propagation
- should remain lightweight

## Transactions

Transactions belong in:
- service orchestration layer

Not:
- handlers
- repositories

## Database

- Use pgx only
- Use raw SQL
- Use Tern migrations
- Never mutate schema manually
- All schema changes require migrations

## Multi-Tenancy

- organization_id propagates through request lifecycle
- future RLS enforcement planned
- tenant safety is infrastructure concern

## Logging

- structured slog logging
- future OpenTelemetry integration planned

---

# Forbidden Patterns

Do NOT:
- introduce ORMs
- introduce global mutable state
- tightly couple providers
- bypass service layer
- place business logic in handlers
- manually mutate DB schema outside migrations

---

# Current Infrastructure

Current implemented systems:
- JWT auth
- provider abstraction
- transaction wrappers
- audit logging
- tenant propagation
- Dockerized PostgreSQL

---

# Current Goal

Current focus:
- observability
- OpenTelemetry
- OpenObserve integration
- production-grade tracing