# Repository Overview

A guided tour of every directory and package in the repository.

---

## Top-Level Structure

```
modular-monolith/
├── cmd/server/           Entry point
├── internal/             All application code (Go convention: unexportable)
├── frontend/             SvelteKit UI (embedded into binary)
├── migrations/           SQL migration files
├── deployments/          Infrastructure configs (Docker, Traefik)
├── scripts/              Automation scripts
├── pkg/                  Shared utilities (importable by other projects)
├── docs/                 Documentation (you are here)
├── .github/workflows/    CI/CD pipelines
├── Dockerfile            Multi-stage build
├── docker-compose.yml    Full stack orchestration
├── go.mod / go.sum       Go module definition
├── Makefile              Developer commands
├── .env.example          Configuration template
└── AGENT.md              AI agent context
```

---

## cmd/server/

**Contains**: `main.go` — the only file.

**Responsibility**: Create the application, start the HTTP server in a goroutine, listen for shutdown signals (SIGINT, SIGTERM), and trigger graceful shutdown.

**Why it exists**: Go convention — `cmd/<name>` for each executable. Keeps `main` minimal; all logic lives in `internal/`.

---

## internal/

Everything below is unexportable outside this module (Go's `internal` convention).

### internal/app/

**Files**: `app.go`, `start.go`, `shutdown.go`

**Responsibility**: Application lifecycle. `app.New()` initializes all dependencies (config → logger → OTEL → database → migrations → repo → service). `app.Start()` creates the router and HTTP server. `app.Shutdown()` gracefully stops HTTP, OTEL, and the DB pool.

### internal/config/

**Files**: `config.go`, `types.go`, `validate.go`, `secrets.go`, `env.go`

**Responsibility**: Load configuration from environment variables using koanf. Validate required fields. Load `.env` file in development. Load Docker secrets from `/run/secrets/` in production.

### internal/router/

**Files**: `router.go`, `routes.go`

**Responsibility**: `router.go` creates the Chi router, applies global middleware, and mounts the frontend fallback. `routes.go` is the **composition root** — it instantiates all module handlers/services/repositories and registers API routes. This is where cross-module adapters are defined.

### internal/middleware/

**Files**: `requestid.go`, `cors.go`, `logging.go`, `recovery.go`, `security.go`, `bodylimit.go`, `metrics.go`, `metricsauth.go`, `ratelimit.go`, `session_identity.go`, `resolve_membership.go`, `tenant.go`, `auth.go`, `devtoken_test.go`, and tests.

**Responsibility**: HTTP middleware. Each file is a single middleware function. The authentication chain (`session_identity` → `resolve_membership` → `tenant`) establishes identity and tenant context for every protected request.

### internal/context/

**Files**: `keys.go`, `tenant.go`, `identity.go`, `membership.go`, `auth.go`, `helpers.go`

**Responsibility**: Request context accessors. Type-safe getters and setters for values stored in `context.Context` (organization_id, identity, membership, legacy authenticated_user).

### internal/database/

**Files**: `postgres.go`, `migrate.go`, `tenant.go`, `tracing.go`

**Responsibility**: Database infrastructure. `postgres.go` creates the pgxpool. `migrate.go` runs SQL migrations on startup. `tenant.go` provides `WithTenantQuery` — the critical multi-tenancy helper. `tracing.go` implements the pgx Tracer interface for OpenTelemetry.

### internal/repository/

**Files**: `repository.go`, `models.go`, `user_repository.go`, `organization_repository.go`, `audit_repository.go`, `tenant.go`, `tx.go`, and integration tests.

**Responsibility**: Legacy shared repository (predates per-module repos). Handles users, organizations, and audit_logs tables. New modules define their own repositories.

### internal/service/

**Files**: `service.go`, `user_service.go`, `organization_service.go`, `transaction.go`, and tests.

**Responsibility**: Legacy shared service layer. Wraps the shared repository with business logic. New modules define their own services.

### internal/modules/

Each subdirectory is an independent module. See [MODULE_GUIDE.md](MODULE_GUIDE.md) for the standard structure.

| Module | Key Files | Owns Tables |
|--------|-----------|-------------|
| `authflow` | `handler.go`, `session.go`, `oidc.go` | None (stateless sessions in cookies) |
| `identity` | `service.go`, `repository.go`, `models.go` | `identities` |
| `identityresolver` | `resolver.go`, `handler.go` | None (reads `users` + `identities`) |
| `organizations` | `handler.go`, `dashboard.go` | Uses shared `organizations` table |
| `users` | `handler.go`, `token.go` | Uses shared `users` table |
| `rbac` | `handler.go`, `service.go`, `repository.go`, `middleware.go` | `roles`, `permissions`, `role_permissions`, `user_roles` |
| `billing` | `handler.go`, `service.go`, `repository.go`, `webhook.go`, `api_handler.go` | `subscriptions` |
| `usage` | `repository.go`, `adapter.go` | `usage_records` |
| `entitlements` | `service.go` | None (reads billing + usage) |
| `onboarding` | `handler.go`, `service.go` | None (orchestrates org + user + rbac) |
| `invitations` | `handler.go`, `service.go`, `repository.go` | `invitations` |
| `auditmod` | `handler.go` | Uses shared `audit_logs` table |
| `health` | `handler.go` | None (DB ping) |

### internal/auth/

**Files**: `middleware.go`

**Responsibility**: Legacy RBAC middleware (superseded by `modules/rbac/middleware.go`). Contains `RequirePermission` that predates the module system.

### internal/audit/

**Files**: `service.go`, `event.go`

**Responsibility**: Audit logging service. Accepts `audit.Event` structs and persists them via the shared repository. Used by all modules that emit audit events.

### internal/platform/payments/

**Files**: Interface definition and Dodo Payments implementation.

**Responsibility**: Payment provider abstraction. `payments.Provider` interface with `CreateCheckoutSession`, `VerifyWebhookSignature`, `ParseWebhookEvent`. `dodo/` implements this for Dodo Payments.

### internal/providers/identity/

**Files**: `zitadel.go`, `oidc.go`

**Responsibility**: OIDC provider implementation. Wraps Zitadel-specific OIDC discovery, token exchange, and ID token verification.

---

## frontend/

```
frontend/
├── src/
│   ├── routes/       SvelteKit page routes
│   ├── lib/          Shared components and utilities
│   ├── app.html      HTML shell
│   └── app.d.ts      TypeScript declarations
├── static/           Static assets (favicon)
├── build/            Output directory (gitignored, populated by pnpm build)
├── embed.go          Go file with //go:embed directive
├── package.json      Dependencies and scripts
├── vite.config.ts    Vite configuration
├── svelte.config.js  SvelteKit config (adapter-static)
└── components.json   shadcn-svelte configuration
```

**Technology**: SvelteKit 5, Svelte 5, TypeScript, Tailwind CSS 4, shadcn-svelte, Vite 8.

**Build**: `pnpm build` → static HTML/JS/CSS in `build/`.

**Embedding**: `embed.go` uses `//go:embed all:build` to include the entire build directory in the Go binary.

---

## migrations/

Sequential SQL files: `001_init.sql` through `016_identity_membership_bridge.sql`.

**Naming**: `NNN_description.sql` (zero-padded 3-digit number).

**Convention**: Up migration above the `---- create above / drop below ----` separator. Down migration below (only up runs in production).

**Ownership**: Each migration corresponds to a feature/module. Tables, indexes, RLS policies, and seed data are all managed here.

---

## deployments/

```
deployments/
├── docker/
│   ├── docker-compose.yml    Production-ready compose (extended)
│   └── .env.example          Production env template
└── traefik/
    └── traefik.yml           Traefik static configuration
```

**Purpose**: Production deployment configurations separate from the development docker-compose.yml at the root.

---

## scripts/

```
scripts/
├── deploy.sh       Full deployment pipeline (validate → build → deploy → health check)
├── backup.sh       PostgreSQL backup to file
└── restore.sh      PostgreSQL restore from backup
```

---

## pkg/

Packages here are importable by external projects (public API).

| Package | Responsibility |
|---------|---------------|
| `pkg/errors` | Sentinel errors and PostgreSQL error codes |
| `pkg/logger` | slog logger factory, PII redaction |
| `pkg/validator` | Input validation (email, required fields) |
| `pkg/response` | HTTP response helpers (OK, Created, Error, BadRequest) |
| `pkg/otel` | OpenTelemetry TracerProvider initialization |

---

## .github/workflows/

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `ci.yml` | Push to master, PRs | Backend (format, vet, test -race, build), Frontend (check, lint, build), Docker (compose config, build image) |
| `release.yml` | Tag push (v*) | Build multi-stage image, push to ghcr.io with semver + SHA tags |

---

## Configuration Files

| File | Purpose |
|------|---------|
| `go.mod` | Go module (github.com/avijitnpm/modular-monolith), Go 1.25, all dependencies |
| `Makefile` | Developer commands (dev, build, test, frontend, docker-*) |
| `Dockerfile` | Multi-stage build (Node → Go → Alpine runtime) |
| `docker-compose.yml` | Full development stack |
| `.env.example` | All configuration variables with development defaults |
| `.dockerignore` | Excludes .git, node_modules, .env from build context |
| `.gitignore` | Standard Go + Node ignores |
| `AGENT.md` | AI coding agent context |


---

## Current Technical Debt

Documented as known architectural evolution, not bugs. These exist because the system evolved from a layered monolith to a modular monolith.

### Legacy Shared Repository (`internal/repository/`)

**What**: A single `Repository` struct owns `users`, `organizations`, and `audit_logs` tables.

**Why it exists**: Created before the per-module pattern was established. Early modules (users, organizations) used a shared data layer.

**Impact**: The shared repository is passed to the shared service, which is passed through the App struct. Newer modules (billing, rbac, invitations) correctly own their own repositories.

**Migration path**: Extract `users` into a dedicated module repository. Extract `organizations` similarly. The shared repository would then only contain cross-cutting infrastructure (audit logging helper).

### Legacy Shared Service (`internal/service/`)

**What**: A `Service` struct wraps the shared repository with business logic for users and organizations.

**Why it exists**: Same evolution as above — predates module-level services.

**Impact**: Still used by the `users` handler and `organizations` handler. Newer modules have self-contained services.

**Migration path**: Move service logic into respective module services. The shared service can then be removed.

### AuthenticatedUser Context (legacy compatibility)

**What**: `internal/context/auth.go` defines `AuthenticatedUser` struct with comments marking all fields as legacy.

**Why it exists**: The original auth middleware populated this struct. The system now uses `Identity` + `Membership` context types. The legacy struct remains for backward compatibility with the `rbac/handler.go` which still reads `AuthenticatedUser`.

**Migration path**: Update `rbac/handler.go` to use `Identity`/`Membership` context types, then remove `AuthenticatedUser`.

### `internal/auth/middleware.go` (dead code)

**What**: Contains a `RequirePermission` middleware that is superseded by `internal/modules/rbac/middleware.go`.

**Why it exists**: Was the original RBAC middleware before rbac became a module.

**Impact**: Not referenced in route registration. Can be deleted.

### Migration numbering gaps

**What**: 16 migrations, some addressing schema corrections (006, 007 relax constraints).

**Why it exists**: Normal schema evolution during development. Constraints were tightened then relaxed as requirements clarified.

**Impact**: None. Migration history is append-only and the system handles them correctly.
