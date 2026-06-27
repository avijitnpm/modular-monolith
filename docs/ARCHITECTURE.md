# Architecture

This is the canonical architecture document for the modular-monolith repository. Read this first.

---

## Design Philosophy

This system is a **multi-tenant SaaS platform** built as a modular monolith. It prioritizes:

1. **Tenant isolation by default** — every tenant-scoped query passes through PostgreSQL Row Level Security. There is no opt-out.
2. **Module boundaries without network overhead** — modules communicate through Go interfaces, not HTTP. Deployment is a single binary.
3. **Embedded frontend** — the SvelteKit UI compiles to static assets and ships inside the Go binary via `go:embed`. One artifact serves both API and UI.
4. **Operational simplicity** — one Docker image, one database, auto-running migrations, structured observability from day one.

---

## High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Internet                                 │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │      Traefik        │  TLS termination
                    │   (reverse proxy)   │  HTTPS redirect
                    └──────────┬──────────┘  Security headers
                               │
                    ┌──────────▼──────────┐
                    │    Go Application   │  :8080
                    │                     │
                    │  ┌───────────────┐  │
                    │  │   Chi Router  │  │
                    │  │  + Middleware │  │
                    │  └───────┬───────┘  │
                    │          │          │
                    │  ┌───────▼───────┐  │
                    │  │    Modules    │  │
                    │  │  (handlers,   │  │
                    │  │   services,   │  │
                    │  │   repos)      │  │
                    │  └───────┬───────┘  │
                    │          │          │
                    │  ┌───────▼───────┐  │
                    │  │   Embedded    │  │
                    │  │   Frontend    │  │  SvelteKit static
                    │  └───────────────┘  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
   ┌──────────▼───┐  ┌────────▼────┐  ┌────────▼────────┐
   │  PostgreSQL  │  │ OpenObserve │  │ Dodo Payments   │
   │  (pgx pool)  │  │   (OTLP)   │  │   (webhooks)    │
   └──────────────┘  └─────────────┘  └─────────────────┘
```

---

## Request Lifecycle

### HTTP Flow

```
Client Request
    │
    ▼
Traefik (TLS, HSTS, CSP, compression)
    │
    ▼
Chi Router
    │
    ├── /health/*          → health module (no auth)
    ├── /metrics           → Prometheus handler (token-protected)
    ├── /api/v1/*          → API routes (see below)
    └── /* (fallback)      → Embedded frontend (SPA)
```

### Middleware Chain (global)

Applied to every request in order:

```
RequestID → CORS → OTEL (if enabled) → Logging → Recovery → Security → BodyLimit → Metrics
```

### Protected Route Chain

For `/api/v1` protected endpoints:

```
SessionIdentityMiddleware → ResolveMembershipMiddleware → TenantContext → [RequirePermission] → Handler
```

1. **SessionIdentityMiddleware** — reads encrypted session cookie, extracts identity (identity_id, provider_id, email, name). Returns 401 if missing.
2. **ResolveMembershipMiddleware** — queries the database to find the user's membership (user record) for their default organization. Sets `MembershipContext` with membership_id + organization_id.
3. **TenantContext** — extracts organization_id from MembershipContext and sets it in request context. All downstream queries use this.
4. **RequirePermission** (per-route) — checks RBAC: does this user's role in this organization include the required permission?

### Handler → Service → Repository Flow

```
Handler (HTTP concerns only)
    │  Decode request, validate input
    │  Extract org_id from context
    ▼
Service (business logic)
    │  Orchestrate operations
    │  Enforce business rules
    │  Emit audit events
    ▼
Repository (data access)
    │  database.WithTenantQuery(pool, ctx, orgID, func(tx) { ... })
    │  SET LOCAL app.current_organization_id = orgID
    │  Execute SQL within transaction
    ▼
PostgreSQL (RLS enforced)
```

---

## Module Architecture

### Module List

| Module | Responsibility |
|--------|---------------|
| `authflow` | OIDC login/callback/logout, session management |
| `identity` | Identity records (maps external provider users to internal identities) |
| `identityresolver` | Resolves identity → membership, lists memberships |
| `organizations` | Organization CRUD, dashboard aggregation |
| `users` | User/membership registration within an org |
| `rbac` | Roles, permissions, user-role assignments |
| `billing` | Subscriptions, checkout sessions, webhook processing |
| `usage` | Per-organization usage tracking and metering |
| `entitlements` | Plan-based feature gating (combines billing + usage) |
| `onboarding` | First-time org+user+role creation flow |
| `invitations` | Invite users to join an organization |
| `auditmod` | Audit log HTTP handler |
| `health` | Liveness and readiness probes |

### Module Structure (standard)

```
internal/modules/<module>/
    handler.go       HTTP handlers
    service.go       Business logic, interfaces
    repository.go    Database access (pgxpool + WithTenantQuery)
    models.go        Domain types
    dto.go           Request/response types
    errors.go        Module-specific sentinel errors
    middleware.go    Module-specific middleware (optional, e.g. rbac)
    *_test.go        Tests
```

### Module Boundaries

Modules do NOT import each other's repositories directly. Cross-module communication uses:
- **Adapter interfaces** defined in `routes.go` — small interfaces that wrap another module's service
- **Dependency injection** at route registration time

---

## Database Architecture

### PostgreSQL with pgx/v5

- Connection pool: `pgxpool.Pool` (connection string from `DATABASE_URL`)
- Tracing: pgx tracer injects OpenTelemetry spans into every query when OTEL is enabled
- No ORM — raw SQL with parameterized queries

### Multi-Tenancy Strategy

Every tenant-scoped table has an `organization_id TEXT NOT NULL` column with:
1. RLS enabled: `ALTER TABLE <table> ENABLE ROW LEVEL SECURITY`
2. RLS policy: `USING (organization_id = current_setting('app.current_organization_id', true))`
3. WITH CHECK clause for INSERT/UPDATE

The `database.WithTenantQuery` helper:
1. Begins a transaction
2. Executes `SELECT set_config('app.current_organization_id', $1, true)` — sets the GUC variable for this transaction only (`is_local=true`)
3. Runs the caller's function
4. Commits (or rolls back on error)

**Every query against tenant-scoped data MUST go through `WithTenantQuery`**. Direct pool queries bypass RLS.

### Tables with RLS

- `users` — memberships per org
- `organizations`
- `roles`, `role_permissions`, `user_roles` — RBAC per org
- `subscriptions` — billing per org
- `audit_logs` — audit trail per org
- `usage_records` — metering per org

### Tables WITHOUT RLS (global)

- `permissions` — system-wide permission definitions
- `identities` — cross-organization identity records
- `schema_migrations` — migration tracking

### Migration System

- Files: `migrations/NNN_description.sql` (zero-padded sequential numbers)
- Runs on startup before the HTTP server starts
- Tracks applied versions in `schema_migrations` table
- Supports `---- create above / drop below ----` separator for up/down portions (only "up" runs)

---

## Authentication Flow

```
Browser                    App                         Zitadel (OIDC)
   │                        │                              │
   │  GET /api/v1/auth/login│                              │
   │───────────────────────>│                              │
   │                        │  Generate state + nonce      │
   │                        │  Store in cookie             │
   │  302 → authorize URL   │                              │
   │<───────────────────────│                              │
   │                        │                              │
   │  User authenticates    │                              │
   │────────────────────────────────────────────────────-->│
   │                        │                              │
   │  GET /callback?code=…  │                              │
   │───────────────────────>│                              │
   │                        │  Exchange code → tokens      │
   │                        │─────────────────────────────>│
   │                        │  Verify ID token             │
   │                        │  FindOrCreateIdentity()      │
   │                        │  Set encrypted session cookie│
   │  302 → / or /onboarding│                              │
   │<───────────────────────│                              │
```

- Sessions are encrypted cookies (AES-GCM using `SESSION_SECRET`)
- Session contains: identity_id, provider subject, email, name
- No JWT tokens for API auth — session cookies only
- Development mode: optional dev token endpoint for testing

---

## Authorization Flow

```
Request
    │
    ▼
TenantContext middleware (org_id in context)
    │
    ▼
RequirePermission("billing.read") middleware
    │
    ├── Get membership_id from MembershipContext
    ├── Query: users → user_roles → role_permissions → permissions
    ├── WHERE org_id AND user_id AND permission_name
    │
    ├── allowed → proceed to handler
    └── denied  → 403 Forbidden
```

Permissions are string-based: `users.read`, `billing.write`, `audit.read`, `settings.write`, etc.

Default roles bootstrapped per organization: `owner` (all), `admin` (all except billing.write), `member` (users.read, settings.read), `viewer` (read-only subset).

---

## Billing Architecture

- **Provider**: Dodo Payments (configurable via `payments.Provider` interface)
- **Flow**: Create checkout session → redirect user → webhook confirms subscription
- **Storage**: `subscriptions` table with RLS (one active subscription per org)
- **Webhook**: signature-verified, idempotent (upsert by provider_subscription_id)
- **Entitlements**: combine subscription plan + usage to determine feature access

---

## Observability

### Tracing

- OpenTelemetry SDK initialized at startup
- OTLP/HTTP exporter sends to OpenObserve
- pgx database tracer instruments every SQL query
- otelhttp middleware instruments every HTTP request with route-level span names
- If OpenObserve is unavailable, the exporter operates asynchronously and the app continues

### Metrics

- Prometheus client_golang
- Custom `middleware.Metrics` records request count, duration, status codes
- `/metrics` endpoint protected by bearer token (`METRICS_TOKEN`)

### Logging

- `log/slog` (structured JSON in production)
- PII redaction in logger (emails, tokens)
- Request ID propagation via middleware

---

## Docker Architecture

### Dockerfile (multi-stage)

```
Stage 1: node:22-alpine      → Build SvelteKit (pnpm build)
Stage 2: golang:1.25-alpine  → Build Go binary (CGO_ENABLED=0, -trimpath, -ldflags="-s -w")
Stage 3: alpine:3.21.3       → Runtime (non-root user 10001, read-only FS, healthcheck)
```

### Docker Compose Services

| Service | Image | Purpose |
|---------|-------|---------|
| `app` | modular-monolith-app:local | Application (read-only, non-root, 512MB limit) |
| `postgres` | postgres:17.5-alpine | Database (1GB limit, healthcheck) |
| `openobserve` | openobserve/openobserve:v0.14.5 | Traces/logs (512MB limit) |
| `traefik` | traefik:v3.4.0 | Reverse proxy (128MB limit) |

All services on internal bridge network. Only Traefik exposes :80/:443 publicly.

### Security Hardening

- `no-new-privileges:true` on all containers
- `read_only: true` on app container (tmpfs for /tmp)
- Non-root user (UID 10001)
- Resource limits and reservations
- Log rotation (10MB × 3 files)
- HSTS, CSP, X-Frame-Options via Traefik labels

---

## Deployment Architecture

### CI/CD Pipeline (GitHub Actions)

```
Push to master / PR
    │
    ├── Backend job:  gofmt → go vet → go test -race → go build
    ├── Frontend job: pnpm install → svelte-check → eslint → pnpm build
    └── Docker job:   docker compose config → docker build

Tag push (v*)
    └── Release job: docker build → push to ghcr.io
```

### Production Deployment

1. `./scripts/deploy.sh` — validates env, builds, deploys, health-checks
2. App starts → migrations run → HTTP server starts
3. Traefik routes HTTPS to app
4. Let's Encrypt for TLS (HTTP challenge)

---

## Frontend Embedding Strategy

```
frontend/
├── src/              SvelteKit 5 app (Svelte, TypeScript, Tailwind, shadcn-svelte)
├── build/            Static output (adapter-static)
├── embed.go          //go:embed all:build
└── package.json      pnpm + vite

At compile time:
  1. pnpm build → outputs static HTML/JS/CSS to frontend/build/
  2. go build → embeds frontend/build/ into binary via go:embed
  
At runtime:
  - Router fallback: non-/api/ requests → frontend.Handler()
  - frontend.Handler() serves embedded static files
  - SPA routing: unknown paths serve index.html
```

---

## Startup Sequence

```
main()
  │
  ├── app.New()
  │     ├── config.Load()         Load .env + env vars via koanf
  │     ├── logger.New()          slog.Logger (JSON in production)
  │     ├── otel.Init()           TracerProvider + OTLP exporter
  │     ├── database.New()        pgxpool.Pool + pgx tracer
  │     ├── database.Migrate()    Run pending SQL migrations
  │     ├── repository.New()      Central repository struct
  │     └── service.New()         Service layer
  │
  ├── app.Start() [goroutine]
  │     ├── router.New()          Chi + middleware + routes + frontend
  │     └── http.Server.ListenAndServe(:8080)
  │
  └── signal.Notify(SIGINT, SIGTERM)
        └── app.Shutdown()
              ├── http.Server.Shutdown(10s)
              ├── otel.Shutdown(10s)
              └── pgxpool.Close()
```

---

## Package Dependency Rules

```
cmd/server      → internal/app
internal/app    → internal/config, database, repository, service, router, audit
internal/router → internal/modules/*, internal/middleware, internal/service, frontend
internal/modules/* → internal/database (WithTenantQuery), internal/context, pkg/*
internal/modules/* → NEVER import other modules directly
pkg/*           → standard library only (no internal imports)
```

Modules communicate through interfaces and adapters, never through direct package imports.


---

## Intentionally Not Implemented

The following systems were deliberately excluded. This section exists so future engineers understand what was considered and rejected.

| System | Why Not |
|--------|---------|
| **Microservices** | Single team, no independent scaling need. Network overhead, distributed tracing complexity, and deployment orchestration would dominate development time. See ADR-003. |
| **Event Bus / Message Queue** | No asynchronous workflows exist. All operations are request-response. Adding Kafka/RabbitMQ would introduce eventual consistency without a use case. |
| **CQRS** | Read and write patterns are symmetric. Single PostgreSQL with RLS handles both adequately. Separate read/write models would double schema maintenance. |
| **Event Sourcing** | Audit logging provides event history. Full event sourcing adds immense complexity (projections, snapshots, replay) with no benefit for CRUD-heavy SaaS. |
| **GraphQL** | See ADR-016. Purpose-built REST endpoints serve known frontend views. No over-fetching problem exists. |
| **gRPC** | No service-to-service communication exists. Single binary has no RPC boundary. gRPC would add protobuf compilation, code generation, and HTTP/2 complexity. |
| **Redis** | No caching layer needed. PostgreSQL handles current load. Session state is in encrypted cookies (stateless). Adding Redis would introduce a failure mode without solving a real problem. |
| **Kafka** | No event streaming use case. Billing webhooks are processed synchronously with idempotent upserts. If webhook volume grows, this decision can be revisited. |
| **Background Workers** | All operations complete within the HTTP request lifecycle. Migrations run at startup. No scheduled jobs exist. If needed, a goroutine-based worker within the binary is the first step — not an external queue. |
| **Kubernetes** | Docker Compose meets deployment needs. K8s operational overhead (networking, secrets, RBAC, helm charts, ingress controllers) is unjustified for a single-service deployment. |
| **ORM** | See ADR-011. Raw SQL gives full control over queries. ORMs hide behavior and make RLS integration harder. |
| **API Gateway** | Traefik handles TLS and routing. The Go application handles auth, rate limiting, and CORS. No separate API gateway layer is needed. |

These decisions should be revisited only when a concrete scaling problem or business requirement demands them — not preemptively.
