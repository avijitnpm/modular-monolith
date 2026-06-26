# Bootstrap Audit

Infrastructure audit of the modular-monolith repository.

---

## Current Architecture

```
┌─────────────────────────────────────────────────────┐
│  docker compose up -d                               │
├─────────────┬───────────┬────────────┬──────────────┤
│  postgres   │ openobserve│    app     │   traefik    │
│  :5432      │  :5080    │   :8080    │  :80 → :443  │
│  (healthy)  │ (started) │ (healthy)  │ (after app)  │
└─────────────┴───────────┴────────────┴──────────────┘

Dockerfile (multi-stage):
  Stage 1: node:22-alpine → pnpm build → frontend/build/
  Stage 2: golang:1.25-alpine → go build (with embedded frontend)
  Stage 3: alpine:3.21.3 → runtime (server + migrations)
```

---

## Bootstrap Sequence

From a completely fresh machine:

```bash
git clone <repo-url> && cd modular-monolith
cp .env.example .env
docker compose up -d
```

What happens automatically:
1. Dockerfile stage 1 builds frontend (pnpm install + pnpm build)
2. Dockerfile stage 2 embeds frontend/build/ into Go binary
3. Dockerfile stage 2 compiles Go server
4. PostgreSQL starts with healthcheck (`pg_isready`)
5. OpenObserve starts (no healthcheck — distroless image)
6. App starts after postgres is healthy
7. App runs migrations automatically (`internal/database/migrate.go`)
8. App reports healthy on `/health/ready`
9. Traefik starts after app is healthy

**Verification:**

```bash
curl http://localhost:8080/health/ready
# {"status":"ready"}
```

---

## CI Bugs (Fixed)

### All GitHub Actions used tag references instead of commit SHAs

**Problem:** Organization policy requires all actions pinned to full-length commit SHAs.

**Root cause:** Workflows used mutable tag references (`@v4`, `@v5`, `@v3`, `@v6`).

**Fix applied:**

| Action | SHA |
|--------|-----|
| `actions/checkout@v4` | `34e114876b0b11c390a56381ad16ebd13914f8d5` |
| `pnpm/action-setup@v4` | `b906affcce14559ad1aafd4ab0e942779e9f58b1` |
| `actions/setup-node@v4` | `49933ea5288caeca8642d1e84afbd3f7d6820020` |
| `actions/setup-go@v5` | `40f1582b2485089dde7abd97c1529aa768e1baff` |
| `docker/login-action@v3` | `c94ce9fb468520275223c153574b00df6fe4bcc9` |
| `docker/metadata-action@v5` | `c299e40c65443455700f0fdfc63efafe5b349051` |
| `docker/build-push-action@v6` | `10e90e3645eae34f1e60eeb005ba3a3d33f178e8` |

All include `# vN` comments for human readability.

---

## Docker Bugs

**None found.** Docker bootstrap is fully automatic:

- ✅ Dockerfile builds frontend and backend in one pass
- ✅ No pre-existing `frontend/build/` required (git-ignored)
- ✅ Migrations run on app startup (no manual `tern migrate`)
- ✅ `depends_on` conditions are correct:
  - postgres: `service_healthy` (has `pg_isready` healthcheck)
  - openobserve: `service_started` (distroless — cannot run healthcheck)
  - app: `service_healthy` (for Traefik dependency)
- ✅ Docker healthcheck uses `/health/live` (liveness, no DB check)
- ✅ Compose healthcheck uses `/health/ready` (readiness, includes DB check)
- ✅ `packageManager: "pnpm@10.22.0"` is set in `frontend/package.json`

---

## Traefik Bugs

### HTTP→HTTPS redirect is production-only and intentional

**Behavior:** Port 80 permanently redirects to port 443.

**Why:** `deployments/traefik/traefik.yml` configures:
```yaml
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
          permanent: true
```

**Impact on localhost:** Traefik will start and attempt ACME certificate issuance via Let's Encrypt. This will **fail** for `localhost` (cannot issue certs for localhost). HTTPS on port 443 will use Traefik's default self-signed certificate.

**This is NOT a bug.** It is intentional production configuration.

**Local development access:**

The app is directly accessible on port 8080 without Traefik:
```bash
curl http://localhost:8080/health/ready
curl http://localhost:8080/
```

Traefik is only relevant for production (real domain with valid TLS).

**If you need Traefik locally with self-signed cert:**
```bash
curl -k https://localhost/health/ready
```

### Docker healthchecks bypass Traefik (correct)

- Dockerfile HEALTHCHECK: `wget -qO- http://127.0.0.1:8080/health/live`
- Compose app healthcheck: `wget -qO- http://127.0.0.1:8080/health/ready`

Both hit the app directly on port 8080 inside the container. They do not go through Traefik. This is correct — healthchecks should never depend on the reverse proxy.

---

## Frontend Embedding

**No bugs found.**

| Path | Builds frontend? |
|------|-----------------|
| Docker (`docker compose up -d`) | ✅ Dockerfile stage 1 |
| CI backend job | ✅ `pnpm install && pnpm build` before `go build` |
| CI frontend job | ✅ Independent build/lint/check |
| CI docker job | ✅ Full `docker build` |
| Release workflow | ✅ `docker/build-push-action` uses Dockerfile |
| Local dev (`make frontend-build`) | ✅ `pnpm install && pnpm build` |

`frontend/build/` is in `.gitignore` — it is never expected to exist in git.

---

## Authentication Setup

### Architecture

- **Provider:** Zitadel (external OIDC identity provider)
- **Not in docker-compose:** Correct — Zitadel is a separate service
- **Startup dependency:** None — OIDC discovery is **lazy** (first token validation)
- **App boots without Zitadel:** ✅ Yes

### How a fresh developer authenticates

In development mode (`APP_ENV=development`):

1. The app accepts dev tokens signed with `DEV_TOKEN_SECRET`
2. No external Zitadel instance is required for basic operation
3. Health endpoints and frontend are served without authentication

For full OIDC flow, a developer needs an external Zitadel instance configured with:
- `OIDC_ISSUER` pointing to the Zitadel URL
- `OIDC_CLIENT_ID` matching the Zitadel application
- `OIDC_REDIRECT_URL` matching the configured callback

### Should docker-compose include Zitadel?

**No.** Zitadel is a complex service (needs CockroachDB/PostgreSQL of its own). Including it would:
- Add 2+ GB memory requirement
- Add 30+ seconds to startup
- Add configuration complexity

The current approach is correct:
- Dev tokens for local development
- External Zitadel for staging/production

### Recommendation

No change needed. The `.env.example` already provides working defaults for development mode. A developer can:
1. `docker compose up -d`
2. Access `http://localhost:8080/` (frontend)
3. Use `DEV_TOKEN_SECRET` to generate dev auth tokens for API access

---

## Remaining Risks

| Risk | Severity | Notes |
|------|----------|-------|
| Traefik ACME fails on localhost | Low | Not a bug — use port 8080 directly |
| `SESSION_SECRET` placeholder value | Low | Development-only; production validation rejects it |
| No app port exposed in docker-compose | Info | App is accessible only via Traefik or Docker network. Port 8080 is not published to host. |

### App port not published

The `app` service in `docker-compose.yml` does not have a `ports:` mapping. The app is only accessible:
- Via Traefik (ports 80/443)
- Via Docker internal network
- From inside the container (healthcheck)

For local development verification, add to docker-compose or use:
```bash
docker compose exec app wget -qO- http://127.0.0.1:8080/health/ready
```

This is intentional — Traefik is the single ingress point. But it means `curl http://localhost:8080` from the host will **not work** unless a port mapping is added.

---

## Files Modified

| File | Change |
|------|--------|
| `.github/workflows/ci.yml` | Pinned all actions to commit SHAs |
| `.github/workflows/release.yml` | Pinned all actions to commit SHAs |
