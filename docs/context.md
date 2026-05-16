# Modular Monolith Architecture Journal (Day 1 → Day 10)

## Project Overview

This repository is a backend-first modular monolith architecture built using:

- Go 1.22+
- Chi Router
- PostgreSQL 17
- pgx
- Tern migrations
- SvelteKit (future frontend)
- Zitadel (future auth provider)
- Dodo Payments (future billing provider)
- OpenTelemetry + OpenObserve (future observability)

The architecture follows:

- interface-first design
- dependency injection
- repository-service-handler separation
- infrastructure ownership
- explicit lifecycle management
- modular monolith principles

The project intentionally avoids:
- magic frameworks
- ORMs
- hidden abstractions
- global state
- tightly coupled services

The purpose of this repo is to become:
- a reusable SaaS backend foundation
- AI-agent friendly architecture
- production-ready monolith template

---

# DAY 1 — PROJECT FOUNDATION

## Goal

Initialize the repository and establish foundational project structure.

## Actions Taken

- Initialized git repository
- Created private GitHub repository
- Ran:

```bash
go mod init github.com/avijitnpm/modular-monolith
```

## Important Learnings

### Go Module Namespace

Go imports are tied directly to:
- the `go mod init` module name

Therefore internal imports use:

```go
github.com/avijitnpm/modular-monolith/internal/config
```

instead of relative imports.

---

# DAY 2 — CONFIGURATION SYSTEM

## Goal

Create centralized runtime configuration architecture.

## Files Added

```text
internal/config/
├── config.go
├── env.go
├── validate.go
└── types.go
```

## Concepts Introduced

### Typed Configuration

Instead of:

```go
os.Getenv("PORT")
```

everywhere in the app,
all runtime values are loaded into typed structs.

Example:

```go
cfg.Server.Port
```

Benefits:
- autocomplete
- validation
- centralized runtime control
- AI readability

---

### Koanf

Used as configuration aggregation library.

Purpose:
- load environment variables
- later support files/flags if needed

---

### godotenv

Used for local development only.

Behavior:
- loads `.env`
- production uses actual environment variables

---

### Fail Fast Validation

Startup validation ensures missing required env vars crash immediately.

Example:

```go
if cfg.Server.Port == "" {
    return errors.New("SERVER_PORT is required")
}
```

This prevents hidden runtime failures.

---

### Important Rule

Only the config package may access:
- env variables
- runtime config sources

No `os.Getenv()` usage outside config package.

---

# DAY 3 — STRUCTURED LOGGING

## Goal

Introduce production-grade structured logging.

## Files Added

```text
pkg/logger/
├── logger.go
├── levels.go
└── context.go
```

## Concepts Introduced

### slog

Used as structured logger.

Reasons:
- Go standard library aligned
- OTEL-friendly
- structured logging support

---

### Environment-Aware Logging

Development:
- text logs

Production:
- JSON logs

Example production log:

```json
{
  "level":"INFO",
  "msg":"server started",
  "port":"8080"
}
```

---

### Structured Logging

Instead of:

```go
log.Println("server running")
```

logs use metadata fields:

```go
logger.Info(
    "server running",
    "port", cfg.Server.Port,
)
```

This makes logs:
- searchable
- machine-readable
- aggregation-friendly

---

### Request-Scoped Logging Foundation

Logger context helpers were added.

Purpose:
- future request IDs
- tenant IDs
- tracing integration

---

### Important Rule

Never log:
- JWTs
- passwords
- secrets
- cookies
- API keys

---

# DAY 4 — ROUTER + MIDDLEWARE PIPELINE

## Goal

Create centralized request lifecycle architecture.

## Technologies Added

- Chi router
- middleware pipeline

## Files Added

```text
internal/router/
├── router.go
└── routes.go

internal/middleware/
├── requestid.go
├── recovery.go
└── logging.go
```

---

## Middleware Pipeline

Request lifecycle became:

```text
Request
↓
Router
↓
Middleware
↓
Handler
↓
Response
```

---

## Middleware Added

### Request ID Middleware

Adds:
- unique UUID per request

Injected into:
- request context
- response headers

Purpose:
- traceability
- production debugging

---

### Recovery Middleware

Recovers panics safely.

Without it:
- panic crashes server

With it:
- panic logged
- 500 returned
- server survives

---

### Logging Middleware

Logs:
- request method
- request path
- duration

---

## Important Learnings

### Stack Traces

Panics produce large stack traces.

These contain:
- Go runtime internals
- stdlib calls
- middleware chain
- exact failure path

They are ugly but extremely valuable.

---

# DAY 5 — APPLICATION CONTAINER + DATABASE FOUNDATION

## Goal

Create centralized infrastructure ownership model.

## Files Added

```text
internal/app/
├── app.go
├── start.go
└── shutdown.go
```

---

## App Container

Created:

```go
type App struct
```

Purpose:
- own runtime dependencies
- centralize lifecycle management

The App container owns:
- config
- logger
- database pool
- repositories
- services

---

## PostgreSQL Connection Pool

Technology:
- pgxpool

Purpose:
- connection reuse
- concurrency handling
- production-safe DB access

---

## Database Lifecycle

Database initialization now:
- uses startup timeout
- validates connectivity using Ping()

---

## Graceful Shutdown

Added:
- OS signal handling
- clean DB shutdown

Handled:
- CTRL+C
- docker stop
- SIGTERM

---

## Important Learnings

Backend systems are mostly:
- lifecycle management
- resource ownership
- graceful startup/shutdown

Not just business logic.

---

# DAY 6 — DATABASE MIGRATIONS

## Goal

Introduce schema versioning and migration lifecycle.

## Technology Added

- Tern migrations

## Structure Added

```text
migrations/
├── tern.conf
├── 001_init.sql
├── 002_users.sql
├── 003_organizations.sql
└── 004_audit_logs.sql
```

---

## Concepts Introduced

### Migrations

Database schema changes became:
- versioned
- reproducible
- deterministic

Migrations behave like:
- git commits for database schema

---

### UUID Architecture

Tables use:

```sql
UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

instead of integers.

Benefits:
- safer APIs
- non-enumerable IDs
- distributed-system friendly

---

### pgcrypto Extension

Installed:
- enables UUID generation

---

### Audit Log Table

Added audit_logs table for:
- future activity tracking
- event history
- compliance logging

---

### Indexes

Added database indexes.

Purpose:
- faster lookups
- scalable queries

---

### Important Rule

Never edit old migrations.

Always create new migrations.

Migration history is immutable.

---

# DAY 7 — REPOSITORY LAYER

## Goal

Separate database access from application logic.

## Files Added

```text
internal/repository/
├── repository.go
├── models.go
├── user_repository.go
└── organization_repository.go
```

---

## Repository Layer

Repositories are responsible ONLY for:
- SQL execution
- row scanning
- DB communication

Repositories should NOT:
- contain business logic
- know HTTP
- contain permissions

---

## Query Architecture

Introduced:
- parameterized SQL

Example:

```sql
VALUES ($1, $2)
```

This prevents:
- SQL injection

---

## Context-Aware Queries

All repository methods accept:

```go
ctx context.Context
```

Purpose:
- request cancellation
- tracing
- timeout propagation

---

## RETURNING Clause

PostgreSQL RETURNING clause introduced.

Purpose:
- retrieve inserted rows immediately
- avoid second query

---

## Important Learning

Repositories are:
- translators between SQL and Go structs

Nothing more.

---

# DAY 8 / DAY 9 — SERVICE LAYER

## Goal

Separate business orchestration from database logic.

## Files Added

```text
internal/service/
├── service.go
├── user_service.go
└── organization_service.go
```

---

## Architecture Evolution

Flow changed from:

```text
route → repository
```

to:

```text
route → service → repository
```

---

## Service Layer Purpose

Services are responsible for:
- workflows
- orchestration
- business rules
- future transactions
- provider coordination

Services should NOT:
- know HTTP
- execute raw SQL
- know routing

---

## Important Learning

Services exist because future workflows become complex.

Example future workflow:

```text
register user
↓
check billing
↓
create audit log
↓
sync metadata
↓
send email
↓
assign organization
```

This belongs in services.

---

# DAY 10 — HANDLERS + JSON API ARCHITECTURE

## Goal

Create proper frontend-ready HTTP API architecture.

## Structure Added

```text
internal/modules/users/
├── handler.go
├── dto.go
└── response.go
```

---

## Handler Layer

Handlers became responsible ONLY for:
- parsing requests
- validation
- calling services
- formatting HTTP responses

Handlers should NOT:
- execute SQL
- contain business workflows

---

## DTOs

DTO = Data Transfer Object.

Used for:
- request payloads
- response payloads

DTOs are API boundary types.

They are NOT:
- DB models
- repository structs

---

## JSON APIs

Introduced:
- JSON request parsing
- JSON responses
- proper HTTP status codes

Example:

```go
json.NewDecoder(r.Body).Decode(&req)
```

Purpose:
- parse frontend JSON payloads

---

## Response Formatting

Responses now:
- set content type
- use JSON encoding
- return proper status codes

Example:

```go
w.WriteHeader(http.StatusCreated)
```

returns:
- HTTP 201 Created

---

## Final Architecture After Day 10

```text
HTTP Request
↓
Router
↓
Middleware
↓
Handler
↓
Service
↓
Repository
↓
PostgreSQL
↓
Response
```

---

# CURRENT ARCHITECTURE STATUS

The repository now includes:

## Infrastructure Layer
- config system
- logging system
- app lifecycle
- middleware
- routing
- database pooling

## Database Layer
- migrations
- repositories
- pgx integration
- schema management

## Application Layer
- services
- handlers
- DTOs
- JSON APIs

---

# IMPORTANT GLOBAL RULES

## Routers
Only register routes.

---

## Handlers
Only handle HTTP concerns.

---

## Services
Only handle business orchestration.

---

## Repositories
Only handle database access.

---

## Config Package
Only package allowed to access env variables.

---

# CURRENT TECHNICAL FLOW

```text
main.go
↓
App Container
↓
Router
↓
Middleware
↓
Handlers
↓
Services
↓
Repositories
↓
PostgreSQL
```

---

# FUTURE PHASES (NOT YET IMPLEMENTED)

Planned next systems:

- validation helpers
- centralized API errors
- Zitadel JWT auth
- tenant middleware
- Row Level Security
- Dodo Payments adapter
- webhook architecture
- OTEL tracing
- OpenObserve integration
- SMTP integration
- SvelteKit embedding
- Traefik deployment
- Docker Compose stack

---

# IMPORTANT ENGINEERING PHILOSOPHY

This project intentionally prioritizes:
- explicit architecture
- composability
- dependency injection
- clean boundaries
- infrastructure ownership
- AI-assisted maintainability

over:
- rapid hacks
- hidden abstractions
- framework magic

The goal is long-term maintainability and reusable SaaS architecture.