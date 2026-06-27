# Architectural Decisions

Engineering rationale for every major technical decision in this repository. Understand the reasoning before changing anything.

---

## Index

| # | Decision | Summary |
|---|----------|---------|
| ADR-001 | [Why Go](#adr-001-why-go) | Single binary, goroutines, stdlib coverage |
| ADR-002 | [Why Chi Router](#adr-002-why-chi-router) | 100% net/http compatible, middleware composition |
| ADR-003 | [Why Modular Monolith](#adr-003-why-modular-monolith-not-microservices) | Module boundaries without network overhead |
| ADR-004 | [Why PostgreSQL](#adr-004-why-postgresql) | Native RLS, mature tooling, ACID |
| ADR-005 | [Why Row Level Security](#adr-005-why-row-level-security) | Database-level tenant isolation |
| ADR-006 | [Why Embedded Frontend](#adr-006-why-embedded-frontend) | Single artifact, no CORS, same-origin cookies |
| ADR-007 | [Why Docker](#adr-007-why-docker-not-bare-metal-or-kubernetes) | Reproducible, single `docker compose up` |
| ADR-008 | [Why OpenTelemetry](#adr-008-why-opentelemetry) | Vendor-neutral observability standard |
| ADR-009 | [Why OpenObserve](#adr-009-why-openobserve-not-jaegergrafana) | Single binary, traces+logs+metrics |
| ADR-010 | [Why Traefik](#adr-010-why-traefik-not-nginxcaddy) | Docker-native, automatic HTTPS, label-based config |
| ADR-011 | [Why Repository Pattern](#adr-011-why-repository-pattern) | Clear persistence boundary, no ORM |
| ADR-012 | [Why Service Layer](#adr-012-why-service-layer-not-handlers-calling-repos-directly) | Testable business logic, composable from multiple callers |
| ADR-013 | [Why Module Ownership](#adr-013-why-module-ownership-not-shared-repositories) | Clear table ownership, no migration conflicts |
| ADR-014 | [Why Mandatory Tenant Isolation](#adr-014-why-tenant-isolation-is-mandatory) | Security boundary, not convenience |
| ADR-015 | [Why Not GraphQL](#adr-015-why-not-graphql) | REST sufficient, no over-fetching problem |
| ADR-016 | [Why Session Cookies](#adr-016-why-session-cookies-not-jwts-for-api-auth) | XSS-immune, instant revocation, no refresh logic |

---

## ADR-001: Why Go

**Problem**: Need a language for a multi-tenant SaaS backend that compiles to a single static binary, handles concurrent connections efficiently, and has a simple deployment model.

**Alternatives considered**:
- **Rust** — Higher performance ceiling but significantly longer development cycles. Ownership semantics are overkill for CRUD-heavy SaaS.
- **TypeScript/Node.js** — Familiar ecosystem but single-threaded event loop, larger Docker images, runtime dependency.
- **Java/Kotlin** — JVM startup time, memory footprint, complex deployment artifacts.

**Why rejected**: Go compiles to a single ~20MB binary with no runtime dependencies. Goroutines handle thousands of concurrent connections without callback complexity. The standard library covers HTTP, JSON, crypto, and testing. Deploy = copy one file.

**Tradeoffs**: More verbose error handling. No generics for domain modeling (acceptable for this scope). Limited ecosystem compared to Node.

---

## ADR-002: Why Chi Router

**Problem**: Need an HTTP router that is idiomatic Go, supports middleware composition, and provides route grouping.

**Alternatives considered**:
- **net/http (stdlib)** — No route parameters, no middleware chaining, no grouping.
- **Gin** — Popular but uses its own context type, not compatible with standard `http.Handler`.
- **Echo** — Similar to Gin, non-standard handler signatures.
- **gorilla/mux** — Now archived/unmaintained.

**Why Chi**: Chi is 100% compatible with `net/http`. Handlers are standard `http.HandlerFunc`. Middleware is standard `func(http.Handler) http.Handler`. No framework lock-in. Route grouping with `.Group()` and `.Route()` maps naturally to access-level separation (public vs. protected).

**Long-term implications**: Any standard Go HTTP middleware works with Chi. Migrating away from Chi would require minimal changes since all handlers use the standard signature.

---

## ADR-003: Why Modular Monolith (not Microservices)

**Problem**: The system has multiple bounded contexts (auth, billing, RBAC, organizations) but a small team and no need for independent scaling.

**Alternatives considered**:
- **Microservices** — Independent deployment, language-per-service flexibility.
- **Layered monolith** — Simpler but boundaries become blurry over time.

**Why rejected**: Microservices add network calls, distributed tracing complexity, deployment orchestration, API versioning, eventual consistency, and a Kubernetes requirement. For a team of 1–5, this overhead dominates feature development time. A layered monolith without enforced boundaries inevitably becomes a big ball of mud.

**Why modular monolith**: Module boundaries are enforced by package structure and dependency rules. Modules cannot import each other's packages — they communicate through interfaces and adapters defined at the composition root (`routes.go`). This gives the separation benefits of microservices without network overhead, while the single deployment model means one Docker image, one database, and zero inter-service networking.

**When to split**: If a single module requires independent scaling (e.g., billing webhook processing under extreme load), extract it. Until then, resist premature distribution.

---

## ADR-004: Why PostgreSQL

**Problem**: Need a relational database that supports multi-tenancy at the database level, has mature tooling, and handles the schema complexity of RBAC + billing + audit.

**Alternatives considered**:
- **MySQL** — No Row Level Security. Would require application-level filtering.
- **MongoDB** — Schema flexibility unnecessary; multi-tenancy via separate databases is expensive.
- **CockroachDB** — Distributed SQL overkill for single-region SaaS.
- **SQLite** — No concurrent writes; single-node only.

**Why PostgreSQL**: Native Row Level Security (RLS) policies enforce tenant isolation at the database engine level. Even a SQL injection cannot cross tenant boundaries because RLS is enforced before results are returned. PostgreSQL also provides: JSONB for flexible metadata, UUID generation, excellent indexing, ACID transactions, and a massive operations ecosystem.

---

## ADR-005: Why Row Level Security

**Problem**: Multi-tenant system must guarantee that Tenant A can never access Tenant B's data, even if application code has bugs.

**Alternatives considered**:
- **Application-level WHERE clauses** — Every query must remember to filter. One missed clause = data leak.
- **Separate schemas per tenant** — Connection pool explosion, migration complexity multiplied by N tenants.
- **Separate databases per tenant** — Maximum isolation but impossible to manage at scale.

**Why RLS**: Defense in depth. The `tenant_isolation_*` policies on every tenant-scoped table mean that even if a developer writes `SELECT * FROM users`, PostgreSQL returns only rows matching the current session's `app.current_organization_id`. The application sets this GUC variable once per transaction via `SET LOCAL`, and it cannot leak across connections because `is_local=true` scopes it to the transaction.

**Tradeoffs**: Every query must run inside a transaction with the tenant context set (via `WithTenantQuery`). Raw pool queries bypass RLS — this is intentional for cross-tenant operations (e.g., identity resolution) but must be used carefully.

---

## ADR-006: Why Embedded Frontend

**Problem**: Need to serve a web UI alongside the API without managing a separate deployment, CDN, or CORS configuration.

**Alternatives considered**:
- **Separate frontend deployment** (Vercel, Netlify, S3) — Requires CORS, separate CI/CD, environment-specific API URLs, cookie configuration across origins.
- **Server-side rendering** (Go templates) — Limited interactivity, no component ecosystem.

**Why embedded**: `go:embed` compiles the static SvelteKit build into the Go binary. One artifact serves both API and UI. No CORS needed (same origin). Session cookies work without SameSite complexity. Deployment is deploying one container. The router serves `/api/*` from handlers and everything else from the embedded filesystem (SPA fallback to index.html).

**Tradeoffs**: Frontend changes require a full rebuild. No CDN edge caching. Acceptable for B2B SaaS where UI is behind authentication anyway.

---

## ADR-007: Why Docker (not bare metal or Kubernetes)

**Problem**: Need reproducible deployments that work identically in development and production.

**Alternatives considered**:
- **Bare metal / systemd** — No reproducibility. Works-on-my-machine problems.
- **Kubernetes** — Massive operational overhead for a single-service deployment.
- **Serverless** (Lambda, Cloud Run) — Cold starts, connection pooling issues with PostgreSQL, complex local development.

**Why Docker Compose**: Single `docker compose up` starts the entire stack (app, postgres, openobserve, traefik). Same configuration works in development and production. Adding a service = adding a section to docker-compose.yml. Horizontal scaling (when needed) is achievable with Docker Swarm or a simple load balancer in front of multiple app containers.

---

## ADR-008: Why OpenTelemetry

**Problem**: Need distributed tracing and structured observability without vendor lock-in.

**Alternatives considered**:
- **Datadog/New Relic SDK** — Vendor lock-in, expensive at scale.
- **Jaeger client** — Deprecated in favor of OTEL.
- **No tracing** — Debugging production issues becomes guesswork.

**Why OTEL**: Vendor-neutral standard. The `otlptracehttp` exporter sends to any OTEL-compatible backend. Today it's OpenObserve; tomorrow it could be Grafana Tempo, Honeycomb, or Datadog — zero code changes, just reconfigure the endpoint URL.

**Tradeoffs**: OTEL SDK adds ~5ms overhead per request. The exporter is async — if the backend is down, spans are dropped silently. This is acceptable: observability should never break the application.

---

## ADR-009: Why OpenObserve (not Jaeger/Grafana)

**Problem**: Need a lightweight, self-hosted observability backend for development and small-scale production.

**Alternatives considered**:
- **Jaeger** — Traces only. No logs or metrics correlation.
- **Grafana + Tempo + Loki + Prometheus** — Four services to manage. Complex configuration.
- **Elastic APM** — Heavy (JVM), expensive licensing.

**Why OpenObserve**: Single binary, handles traces + logs + metrics. Minimal resource requirements (512MB RAM). Accepts OTLP directly. Good-enough UI for debugging. Replaceable (change OTEL endpoint) if outgrown.

---

## ADR-010: Why Traefik (not Nginx/Caddy)

**Problem**: Need a reverse proxy with automatic HTTPS, Docker-native service discovery, and security headers.

**Alternatives considered**:
- **Nginx** — Manual config, no automatic Let's Encrypt without certbot, no Docker label-based routing.
- **Caddy** — Automatic HTTPS but less Docker-native than Traefik.
- **No reverse proxy** (Go app does TLS) — Reinventing the wheel. No graceful cert rotation.

**Why Traefik**: Docker provider discovers services via labels. Adding HTTPS = one label. Adding security headers = middleware labels. Let's Encrypt integration is built-in with HTTP challenge. Dashboard for debugging. V3 is stable and actively maintained.

---

## ADR-011: Why Repository Pattern

**Problem**: Need a clear boundary between business logic and database access.

**Alternatives considered**:
- **Direct SQL in handlers** — No separation, untestable business logic.
- **ORM (GORM, ent)** — Magic queries, hard to optimize, hides SQL complexity.
- **Query builder (sqlc)** — Good option but adds a code generation step.

**Why manual repository pattern**: Repositories own SQL. Services own logic. Handlers own HTTP. Each layer is independently testable. Raw SQL with pgx means full control over queries, explicit N+1 prevention, and no "magic" to debug. The `WithTenantQuery` helper makes RLS usage consistent across all repositories.

---

## ADR-012: Why Service Layer (not handlers calling repos directly)

**Problem**: Handlers that directly call repositories accumulate business logic, making them untestable and difficult to compose.

**Decision**: Every module has a Service that:
1. Validates business rules
2. Orchestrates multi-step operations
3. Emits audit events
4. Returns domain errors (not HTTP errors)

Handlers are thin — they decode HTTP, call the service, and encode the response. This means services can be called from webhooks, background jobs, or other modules without going through HTTP.

---

## ADR-013: Why Module Ownership (not shared repositories)

**Problem**: If multiple modules write to the same tables, ownership is unclear, migrations conflict, and invariants break.

**Decision**: Each module owns its tables. The `billing` module owns `subscriptions`. The `rbac` module owns `roles`, `permissions`, `role_permissions`, `user_roles`. Cross-module access happens through interfaces, never through shared table access.

**Exception**: The shared `repository` package (`internal/repository`) exists for legacy tables (`users`, `organizations`, `audit_logs`) that predate the module system. New modules must own their tables entirely.

---

## ADR-014: Why Tenant Isolation is Mandatory

**Problem**: A single missed WHERE clause in a multi-tenant system can expose one customer's data to another.

**Decision**: RLS cannot be bypassed by application code. The database enforces isolation regardless of query correctness. This means:
- Every tenant-scoped query runs inside `WithTenantQuery`
- Cross-tenant queries (e.g., identity resolution) use the pool directly — these are explicitly designed and audited
- There is no "admin mode" that disables RLS from the application

This is a security boundary, not a convenience feature. It cannot be relaxed.

---

## ADR-015: Why Not GraphQL

**Problem**: Considered for flexible frontend querying.

**Why rejected**: GraphQL adds query complexity analysis, N+1 problems, schema maintenance, and a client-side query layer. For a B2B SaaS with well-defined views, REST endpoints with purpose-built responses are simpler, faster, and easier to cache. The frontend knows exactly what it needs — there's no over-fetching problem to solve.

---

## ADR-016: Why Session Cookies (not JWTs for API auth)

**Problem**: Need to authenticate API requests from the browser-based frontend.

**Alternatives considered**:
- **JWT in Authorization header** — Requires frontend to store tokens (localStorage = XSS risk, memory = lost on refresh).
- **JWT in httpOnly cookie** — Can't revoke without a blocklist. Token size grows with claims.

**Why session cookies**: Encrypted (AES-GCM) httpOnly cookies are immune to XSS (JavaScript cannot read them). Server-side session state means instant revocation (delete the session). No token refresh logic needed. The cookie is automatically sent with every request — no frontend auth code required.
