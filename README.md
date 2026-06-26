# Modular Monolith

## Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 22+ with pnpm (corepack)

## Quick Start (Docker — recommended)

```bash
git clone <repo-url> && cd modular-monolith
cp .env.example .env
docker compose up -d
```

This builds the app image (including frontend) and starts all services.
Migrations run automatically on app startup.

### Verify

```bash
# App health (direct — bypasses Traefik)
curl http://localhost:8080/health/ready

# App health (via Traefik HTTPS — uses self-signed cert in dev)
curl -k https://localhost/health/live

# Postgres
docker compose exec postgres pg_isready

# OpenObserve UI
curl -s -o /dev/null -w "%{http_code}" http://localhost:5080/web/
```

## Development (local Go + frontend)

```bash
make setup                  # copies .env.example, installs frontend deps
make frontend-build         # build frontend (required before first `go run`)
make dev                    # runs Go server on :8080
make frontend               # runs Vite dev server with HMR
```

## Docker Build (standalone)

```bash
docker build -t modular-monolith-app:local .
```

The Dockerfile is fully self-contained with multi-stage build (Node.js → Go → Alpine runtime).

## Makefile Targets

| Target | Description |
|--------|-------------|
| `setup` | Copy .env.example, install frontend deps |
| `dev` | Run Go server locally |
| `build` | Compile Go binary to bin/app |
| `frontend` | Vite dev server |
| `frontend-build` | Build frontend for embedding |
| `test` | Run Go tests |
| `docker-build` | Build Docker image |
| `docker-up` | Start all services |
| `docker-down` | Stop all services |
| `migrate` | Show migration info (auto-runs on startup) |

## Architecture Notes

- **Migrations**: Run automatically on app startup via `internal/database/migrate.go`. No external tools needed in production.
- **OpenObserve**: Observability backend. The OTEL exporter is async — if OpenObserve is unavailable, the app logs a warning and continues with a noop tracer.
- **Frontend**: SvelteKit with adapter-static, embedded into the Go binary via `//go:embed`.
