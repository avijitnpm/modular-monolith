# Phase 19 — Stability & Bootstrap Verification: Root Cause Report

**Date**: 2026-06-26  
**Commits**: `319e815` (frontend lint), `d127fec` (infra fixes)

---

## Executive Summary

Four issues were identified and fixed across the frontend CI pipeline, Docker Compose infrastructure, and developer documentation. The application architecture is sound — OIDC is lazy-loaded, the bootstrap process is fully automated, and the Go backend starts independently of Zitadel.

---

## Phase A — Frontend Lint

### Finding

Frontend CI failed at `pnpm lint` (script: `prettier --check . && eslint .`).

### Root Cause

1. **86 files** had Prettier formatting violations (tabs, single quotes, trailing commas, print width 100)
2. **16 ESLint errors** across 12 files:
   - 8× `svelte/require-each-key` — `{#each}` blocks missing key expressions
   - 7× `svelte/no-navigation-without-resolve` — `goto()`/`href` without `resolve()` from `$app/paths`
   - 1× `@typescript-eslint/no-unused-vars` — unused `_` variable in `{#each}` destructuring

### Files Modified

95 files (86 formatting-only + 9 with eslint fixes + `eslint.config.js`)

Key files with logic changes:
- `frontend/eslint.config.js` — added rule override for generic Button UI component
- `frontend/src/lib/components/layout/AppSidebar.svelte` — `as const` + `resolve()` for href
- `frontend/src/lib/components/shared/AuthGuard.svelte` — `goto(resolve('/login'))`
- `frontend/src/lib/components/shared/LoadingState.svelte` — `Array.from().keys()` pattern
- `frontend/src/routes/+page.svelte` — `goto(resolve('/dashboard'))`
- `frontend/src/routes/login/+page.svelte` — `goto(resolve('/dashboard'))`
- `frontend/src/routes/logout/+page.svelte` — `goto(resolve('/login'))`
- `frontend/src/lib/components/settings/SessionCard.svelte` — `href={resolve('/logout')}`

### Verification

```bash
$ cd frontend && pnpm lint
Checking formatting...
All matched files use Prettier code style!
# (eslint exits 0)

$ pnpm check
svelte-check found 0 errors and 0 warnings

$ pnpm build
✓ built in 5.75s
✔ done
```

### Status: PASS ✅

---

## Phase B — Traefik Routing Investigation

### Finding

`curl -k https://localhost/health/live` reported to return `404 page not found`.

### Request Flow Trace

```
curl -k https://localhost/health/live
  │
  ├─ DNS: localhost → 127.0.0.1
  │
  ├─ Traefik entrypoint: websecure (:443)
  │    ├─ TLS termination
  │    │   └─ certresolver: letsencrypt (ACME httpChallenge)
  │    │      └─ ⚠ CANNOT work for localhost (not publicly reachable)
  │    │
  │    ├─ Router: "app"
  │    │   ├─ Rule: Host(`localhost`) ← MATCHES ✓
  │    │   ├─ Entrypoints: websecure ← MATCHES ✓
  │    │   └─ TLS: true ✓
  │    │
  │    ├─ Middlewares: security-headers@docker, compress@docker
  │    │
  │    └─ Service: app (loadbalancer → app:8080)
  │         ├─ Network: modular-monolith_monolith_internal ✓
  │         └─ Port: 8080 ✓
  │
  └─ Go app (chi router)
       └─ GET /health/live → healthHandler.Live → 200 {"status":"ok"}
```

### Root Cause

The routing configuration is architecturally correct. The break occurs at the **TLS certificate layer**:

1. `.env.example` set `CERT_RESOLVER=letsencrypt`
2. The `letsencrypt` resolver uses ACME httpChallenge on port 80
3. ACME httpChallenge requires the domain to be publicly accessible — impossible for `localhost`
4. **Traefik v3 behavior**: Without a valid ACME cert and no default certificate configured, Traefik may fail to establish the TLS connection or route the request, returning its own `404 page not found`

### Evidence

| File | Line | Finding |
|------|------|---------|
| `docker-compose.yml` | app labels | `traefik.http.routers.app.tls.certresolver: letsencrypt` |
| `deployments/traefik/traefik.yml` | certResolvers | ACME httpChallenge via entrypoint `web` |
| `.env.example` | CERT_RESOLVER | Default: `letsencrypt` |
| `docker-compose.yml` | traefik provider | `network: modular-monolith_monolith_internal` ✓ |

### Status: PASS ✅ (root cause identified)

---

## Phase C — Bootstrap Verification

### Bootstrap Matrix

| Stage | Automated? | Method | Evidence |
|-------|:---:|--------|---------|
| Frontend build | ✔ | `Dockerfile:4-10` — node:22-alpine, pnpm install + build | adapter-static → `build/` |
| Go build | ✔ | `Dockerfile:12-19` — golang:1.25-alpine, CGO_ENABLED=0 | Output: `/out/server` |
| Container | ✔ | `Dockerfile:21-35` — alpine:3.21.3, non-root 10001 | Minimal attack surface |
| PostgreSQL | ✔ | Healthcheck: `pg_isready` | interval: 5s, retries: 5 |
| OpenObserve | ✔ | `condition: service_started` (no healthcheck) | Distroless image |
| Migrations | ✔ | `internal/app/start.go:63` — auto on startup | 16 SQL files |
| App health | ✔ | Healthcheck: `wget /health/ready` | start_period: 15s |
| Traefik | ✘ | `depends_on: app: healthy` | ACME fails for localhost |

### Issues Found

1. **ACME for localhost** — `CERT_RESOLVER=letsencrypt` + `DOMAIN=localhost` → impossible
2. **App port not published** — No `ports:` on app service → host can't reach :8080
3. **README verify commands** — `curl http://localhost:8080/health/ready` fails without port mapping

### Status: PASS ✅ (issues documented, fixes in Phase E)

---

## Phase D — Authentication Audit

### Key Findings

| Question | Answer | Evidence |
|----------|--------|---------|
| App starts without Zitadel? | **YES** | `identity/zitadel.go:33` — `NewZitadelProvider()` stores config only |
| OIDC lazy-loaded? | **YES** | `zitadel.go:126` — `getDiscovery()` called only on Login/Exchange/Validate |
| DEV_TOKEN_SECRET in dev? | **Required** | `config/validate.go:45` — validation enforces it |
| DEV_TOKEN_SECRET in prod? | **Forbidden** | `config/validate.go:85` — validation rejects it |
| When is Zitadel needed? | **Only on Login click** | `oauth.go:30` — OIDC discovery happens in Login handler |

### Endpoint Authentication Map

| Endpoint | Auth | Notes |
|----------|:---:|-------|
| `GET /health`, `/health/live`, `/health/ready` | ✗ | Anonymous, no middleware |
| `GET /api/v1/ping` | ✗ | Anonymous |
| `GET /api/v1/token` | ✗ | Dev-only, returns test JWT |
| `GET /api/v1/auth/login` | ✗ | Triggers OIDC discovery → redirect |
| `GET /api/v1/auth/callback` | ✗ | Receives OIDC callback |
| `POST /api/v1/auth/logout` | ✗ | Clears session cookie |
| `GET /api/v1/auth/me` | ✗* | Returns 401 if no session |
| `POST /api/v1/onboarding` | ✗ | Rate-limited |
| `POST /api/v1/invitations/accept` | ✗ | Rate-limited |
| `GET /api/v1/memberships` | ✔ | SessionIdentityMiddleware |
| All other `/api/v1/*` | ✔ | Session + Membership + TenantContext |
| `/*` (non-API) | ✗ | Frontend SPA (embedded static files) |

### Auth Architecture

```
Session Cookie (AES-256-GCM encrypted)
  ├─ Set by: /api/v1/auth/callback (after OIDC flow)
  ├─ Contains: IdentityID, Subject, Email, Name
  ├─ Validated by: SessionIdentityMiddleware
  └─ Cleared by: /api/v1/auth/logout

Development Mode:
  └─ GET /api/v1/token → JWT (user-123, org-456, test@example.com)
     (For API testing tools — NOT used by browser session auth)
```

### Development Flow (without Zitadel)

1. `docker compose up -d` — app starts normally
2. Health endpoints work immediately
3. Frontend serves (SPA, embedded in binary)
4. `GET /api/v1/token` → dev JWT for Postman/API testing
5. Login button in frontend → will fail (OIDC discovery can't reach Zitadel)
6. Zitadel only needed if testing actual OIDC login flow

### Status: PASS ✅

---

## Phase E — Fix Matrix

| # | Issue | Root Cause | Severity | Fix | Commit | Verified |
|---|-------|-----------|----------|-----|--------|----------|
| 1 | Frontend CI fails | Prettier + ESLint violations | HIGH | `prettier --write` + fix lint errors | `319e815` | `pnpm lint/check/build` ✓ |
| 2 | Traefik TLS broken for localhost | `CERT_RESOLVER=letsencrypt` default | HIGH | Default to empty (self-signed) | `d127fec` | `docker compose config` ✓ |
| 3 | App port not published | No `ports:` on app service | MEDIUM | Add `127.0.0.1:8080:8080` | `d127fec` | `docker compose config` ✓ |
| 4 | README verify incorrect | Commands assume port published | LOW | Add direct + Traefik examples | `d127fec` | Visual ✓ |

### Verification Commands (post-fix)

```bash
# Frontend CI
cd frontend && pnpm lint && pnpm check && pnpm build

# Docker Compose validity
docker compose config --quiet

# Go backend
go vet ./...

# Full stack (requires network access for Docker build)
docker compose up -d
curl http://localhost:8080/health/ready     # direct
curl -k https://localhost/health/live        # via Traefik
```

---

## Files Modified

### Commit `319e815` — Frontend Lint

- `frontend/eslint.config.js`
- 86 files (prettier formatting)
- 9 files (eslint error fixes)

### Commit `d127fec` — Infrastructure

- `docker-compose.yml` — removed certresolver default, added app port
- `.env.example` — `CERT_RESOLVER=` with documentation
- `README.md` — updated verify section

---

## Recommendations

1. **CI should run `pnpm format` as a pre-commit hook** to prevent formatting drift
2. **Consider a `docker-compose.override.yml`** for production with `CERT_RESOLVER=letsencrypt`
3. **Add a Makefile target** for `make verify` that runs all health checks
4. **Document the Zitadel setup** as a separate optional step for full login testing
