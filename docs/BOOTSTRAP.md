# Bootstrap Guide

Canonical startup sequence for the modular-monolith.

## Docker (recommended — zero dependencies beyond Docker)

```bash
git clone <repo-url> && cd modular-monolith
cp .env.example .env
docker compose up -d
```

This single command:
1. Builds the frontend (SvelteKit → static assets)
2. Builds the Go binary with embedded frontend
3. Starts PostgreSQL and waits for it to be healthy
4. Starts OpenObserve (observability)
5. Starts the application (runs migrations automatically)
6. Starts Traefik (reverse proxy with TLS)

### Verify

```bash
# App health (wait ~20s for startup)
curl http://localhost:8080/health/ready

# Postgres
docker compose exec postgres pg_isready

# OpenObserve UI
curl -s -o /dev/null -w "%{http_code}" http://localhost:5080/web/
```

### Tear down

```bash
docker compose down           # stop services, keep data
docker compose down -v        # stop services, remove volumes
```

## Local Development

Prerequisites: Go 1.25+, Node.js 22+, pnpm (via corepack), PostgreSQL running locally.

```bash
# One-time setup
make setup

# Build frontend (required before first go run — go:embed needs the files)
make frontend-build

# Run Go server
make dev

# In a separate terminal: run frontend with HMR
make frontend
```

### Local PostgreSQL

The app connects to `DATABASE_URL` from `.env`. For local dev, update it to point at your local Postgres:

```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable
```

Migrations run automatically when the server starts.

## Migrations

Migrations execute automatically on application startup via `internal/database/migrate.go`.

- SQL files live in `migrations/` (numbered: `001_init.sql`, `002_users.sql`, ...)
- Applied migrations are tracked in a `schema_migrations` table
- No external tools (tern, goose, etc.) are required

To create a new migration:

```bash
make migration name=add_feature_table
```

## Authentication (Zitadel)

The app uses Zitadel as an external OIDC identity provider. Zitadel is NOT part of the Docker Compose stack.

For local development without Zitadel, the app accepts dev tokens via `DEV_TOKEN_SECRET` in `.env`.

For full OIDC flow, configure an external Zitadel instance and set:
- `OIDC_ISSUER`
- `OIDC_CLIENT_ID`
- `OIDC_REDIRECT_URL`

## CI

GitHub Actions runs the same Dockerfile for the Docker build job. The CI pipeline:
1. Backend job: builds frontend → runs Go vet/test/build
2. Frontend job: lint/typecheck/build
3. Docker job: validates compose config, builds the full Dockerfile

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `corepack` fetch error in Docker build | Network issue or missing `packageManager` field | Ensure `frontend/package.json` has `"packageManager": "pnpm@10.22.0"` |
| `go:embed` build error | `frontend/build/` doesn't exist | Run `make frontend-build` first |
| App exits on startup | Database unreachable or migration failure | Check `DATABASE_URL` and Postgres status |
| Traefik not starting | App not healthy yet | Wait for app healthcheck to pass (~15s) |
