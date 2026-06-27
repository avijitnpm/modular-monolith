# Modular Monolith

A multi-tenant SaaS platform built as a modular monolith. Single Go binary, embedded frontend, PostgreSQL with Row Level Security for tenant isolation.

---

## Architecture

Go 1.25 · Chi router · pgx/v5 · SvelteKit 5 · Docker Compose

```
┌──────────────────────────────────────────────────────────┐
│                       Internet                            │
└────────────────────────────┬─────────────────────────────┘
                             │ HTTPS
                  ┌──────────▼──────────┐
                  │       Traefik       │  TLS · HSTS · CSP
                  └──────────┬──────────┘
                             │ :8080
                  ┌──────────▼──────────┐
                  │   Go Application    │
                  │                     │
                  │  Middleware Chain    │
                  │  ───────────────    │
                  │  Feature Modules    │
                  │  (handler→svc→repo) │
                  │  ───────────────    │
                  │  Embedded Frontend  │
                  └───┬─────────┬───────┘
                      │         │
         ┌────────────▼──┐  ┌──▼────────────┐
         │  PostgreSQL   │  │  OpenObserve  │
         │  (RLS)        │  │  (OTLP)       │
         └───────────────┘  └───────────────┘
```

Modules: authflow · identity · organizations · users · rbac · billing · usage · entitlements · onboarding · invitations · audit · health

**Read the full architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| Router | Chi v5 |
| Database | PostgreSQL 17 (pgx/v5 pool) |
| Multi-tenancy | Row Level Security |
| Auth | OIDC (Zitadel) + encrypted session cookies |
| Authorization | RBAC (permissions per role per org) |
| Billing | Dodo Payments |
| Frontend | SvelteKit 5, Tailwind CSS 4, shadcn-svelte |
| Observability | OpenTelemetry → OpenObserve, Prometheus metrics |
| Reverse Proxy | Traefik v3 (TLS, HSTS, CSP) |
| CI/CD | GitHub Actions → GHCR |

---

## Quick Start

### Docker (recommended)

```bash
git clone <repo-url> && cd modular-monolith
cp .env.example .env
docker compose up -d
```

Verify:
```bash
curl http://localhost:8080/health/ready     # direct
curl -k https://localhost/health/live        # via Traefik
```

### Local Development

```bash
make setup              # copies .env, installs frontend deps
make frontend-build     # build frontend (required before first go run)
make dev                # Go server on :8080
make frontend           # Vite dev server with HMR (separate terminal)
```

---

## Repository Layout

```
cmd/server/         Entry point (main.go)
internal/
├── app/            Application lifecycle
├── config/         Configuration loading
├── router/         Chi router + route registration (composition root)
├── middleware/     HTTP middleware chain
├── modules/        Feature modules (handler → service → repository)
├── database/       PostgreSQL pool, migrations, tenant helpers
├── context/        Request context accessors
├── audit/          Audit logging service
├── providers/      External provider implementations (OIDC)
├── platform/       Platform services (payments)
├── repository/     Legacy shared repository
└── service/        Legacy shared service
frontend/           SvelteKit app (embedded via go:embed)
migrations/         Sequential SQL files (run on startup)
deployments/        Docker + Traefik configs
scripts/            Deploy, backup, restore
pkg/                Shared utilities (logger, response, validator, errors, otel)
docs/               Architecture documentation
.github/workflows/  CI + Release pipelines
```

---

## Documentation

| Document | Purpose |
|----------|---------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Complete system architecture — read first |
| [API_REFERENCE.md](docs/API_REFERENCE.md) | HTTP API handbook — every endpoint documented |
| [ARCHITECTURAL_DECISIONS.md](docs/ARCHITECTURAL_DECISIONS.md) | Why every major decision was made |
| [REPOSITORY_OVERVIEW.md](docs/REPOSITORY_OVERVIEW.md) | Guided tour of every directory |
| [MODULE_GUIDE.md](docs/MODULE_GUIDE.md) | How to create a new module |
| [ENGINEERING_RULES.md](docs/ENGINEERING_RULES.md) | Permanent engineering rules |
| [BOOTSTRAP.md](docs/BOOTSTRAP.md) | First-time environment setup |
| [PRODUCTION_CHECKLIST.md](docs/PRODUCTION_CHECKLIST.md) | Pre-production checklist |
| [HARDENING_GUIDE.md](docs/HARDENING_GUIDE.md) | Security hardening details |
| [SECRET_ROTATION.md](docs/SECRET_ROTATION.md) | Secret rotation procedures |

---

## Development Workflow

```bash
make dev              # Run Go server
make frontend         # Run Vite dev server
make test             # Run Go tests (short mode)
make fmt              # Format Go code
make lint             # Run golangci-lint
make build            # Compile Go binary to bin/app
make docker-build     # Build Docker image
make docker-up        # Start Docker Compose stack
make docker-down      # Stop Docker Compose stack
make migration name=X # Create new migration file
```

---

## Contribution Expectations

1. Read [ARCHITECTURE.md](docs/ARCHITECTURE.md) and [ENGINEERING_RULES.md](docs/ENGINEERING_RULES.md) before writing code.
2. Follow the [MODULE_GUIDE.md](docs/MODULE_GUIDE.md) for new features.
3. All changes go through pull requests. CI must pass.
4. Never bypass tenant isolation. Never import across modules directly.
5. Every state-changing operation must emit an audit event.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 22+ with pnpm (corepack)
