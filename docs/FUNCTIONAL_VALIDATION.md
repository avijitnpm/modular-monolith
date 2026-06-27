# Functional Validation Report

**Date**: 2026-06-27
**Validator**: Independent Senior Platform Engineer (zero prior context)
**Method**: Full bootstrap → live system → endpoint-by-endpoint verification

---

## Executive Summary

**Status**: PASS (with 3 bugs found and fixed)

The repository delivers on its architectural promises. A new engineer can clone, run `docker compose up -d`, and have a fully functional multi-tenant SaaS platform running within minutes. Documentation is comprehensive, accurate, and internally consistent. The codebase follows its own engineering rules with high fidelity.

**Overall Confidence**: HIGH

Three bugs were found during live validation — all were implementation errors that violated documented behaviour. All were fixed with minimal changes. No architectural issues or design problems were identified.

---

## Validation Matrix

| Subsystem | Status | Notes |
|-----------|--------|-------|
| Bootstrap | PASS | `cp .env.example .env && docker compose up -d` works. All services start in correct order. |
| Docker | PASS | 4 services (app, postgres, openobserve, traefik). Correct images, resource limits, security hardening. |
| Traefik | PASS | TLS termination, HSTS, CSP, X-Frame-Options via labels. HTTP→HTTPS redirect. Self-signed cert for localhost. |
| Database | PASS | PostgreSQL 17.5-alpine. Named volume persists data. Healthcheck works. |
| Frontend | PASS | SvelteKit builds to static. `go:embed` serves from binary. SPA fallback to index.html works. |
| Backend | PASS | Go 1.25, Chi v5, pgx/v5. Single binary. Compiles cleanly. All tests pass. |
| Authentication | PASS | AES-GCM session cookies. OIDC flow structure correct. Dev token for testing. |
| Identity | PASS | Identity records created/resolved. Cross-org lookup works without RLS. |
| Membership | PASS | Membership resolution from identity_id. MembershipContext set correctly. |
| Tenant | PASS | Organization ID flows through middleware chain to WithTenantQuery. |
| RBAC | PASS | Roles bootstrapped per org (owner, admin, member, viewer). Permission checks enforce access. RequirePermission middleware works. |
| Organizations | PASS | CRUD works. Dashboard aggregation works. Duplicate detection fixed (was Bug #2). |
| Invitations | PASS | Create invitation returns token/URL. Token-based accept flow structured correctly. |
| Billing | PASS | Subscription CRUD works. Checkout session structure correct. Webhook endpoint exists with signature verification. |
| Usage | PASS | Usage counters return correct zero-state. Entitlements compute from plan + usage. |
| Audit | PASS | Audit events emitted for state changes. List endpoint works (after Bug #1 fix). |
| Metrics | PASS | Prometheus endpoint at `/metrics`. Bearer token required (401 without, 200 with). |
| Observability | PASS | OTEL tracing initializes. Async exporter doesn't block app. pgx tracer configured. |
| Persistence | PASS | Data persists across app restarts. Postgres volume persists across `docker compose down`/`up`. Migrations are idempotent. |
| Documentation | PASS | Docs match implementation. 10 comprehensive documents. All architectural claims verified. |

---

## Bugs Found

### Bug #1: Audit Log Endpoint — timestamptz Scan Failure

**Observed**: `GET /api/v1/audit` returns HTTP 500 with `{"error":"failed to list audit logs"}`.

**Expected**: HTTP 200 with audit log entries (per API_REFERENCE.md).

**Root Cause**: `internal/repository/audit_repository.go` scans PostgreSQL `timestamptz` column directly into a Go `string` field (`AuditLog.CreatedAt`). pgx v5 cannot scan `timestamptz` (OID 1184) in binary format into `*string`.

**Files**: `internal/repository/audit_repository.go`

**Fix Applied**: Scan into `time.Time`, then format to `time.RFC3339` string.

**Verified**: Endpoint now returns HTTP 200 with correctly formatted audit entries.

---

### Bug #2: Organization Duplicate Detection — Wrong pgconn Package

**Observed**: Creating a duplicate organization returns HTTP 500 with `{"error":"failed to create organization"}`.

**Expected**: HTTP 409 with `{"error":"organization already exists"}` (per API_REFERENCE.md).

**Root Cause**: `internal/repository/organization_repository.go` imports `github.com/jackc/pgconn` (standalone v1 package) for the `*pgconn.PgError` type assertion. However, pgx v5 returns errors using its own internal `github.com/jackc/pgx/v5/pgconn.PgError` type. The two types have the same name but are different Go types, so `errors.As` with the v1 type never matches.

**Files**: `internal/repository/organization_repository.go`

**Fix Applied**: Changed import from `github.com/jackc/pgconn` to `github.com/jackc/pgx/v5/pgconn` and used `errors.As` for proper error unwrapping.

**Verified**: Duplicate org now returns HTTP 409 with correct error message.

---

### Bug #3: RBAC Handler — Missing Legacy AuthenticatedUser Context

**Observed**: `POST /api/v1/roles` (create role) and `POST /api/v1/users/{id}/roles` (assign role) return HTTP 500 with `{"error":"authenticated user missing"}`.

**Expected**: Role creation/assignment should succeed for users with `settings.write` permission.

**Root Cause**: The RBAC handler's `requestIdentity()` function reads from `appcontext.GetAuthenticatedUser()` — the legacy context type. The current middleware chain (SessionIdentityMiddleware → ResolveMembershipMiddleware → TenantContext) sets `Identity` and `MembershipContext` but never populates the legacy `AuthenticatedUser`. This is documented as technical debt in REPOSITORY_OVERVIEW.md but constitutes a functional break.

**Files**: `internal/middleware/resolve_membership.go`

**Fix Applied**: Added legacy `AuthenticatedUser` context population in `ResolveMembershipMiddleware` when a membership is successfully resolved. This is a backward-compatibility bridge — the proper long-term fix is to update the RBAC handler to use `MembershipContext` directly.

**Verified**: Role creation and assignment now return HTTP 201 with correct responses.

---

## Bugs Fixed Summary

| # | Bug | Severity | Fix | Verified |
|---|-----|----------|-----|----------|
| 1 | Audit log timestamptz scan | High (endpoint completely broken) | Scan into time.Time | ✅ |
| 2 | Org duplicate pgconn package mismatch | Medium (wrong error code returned) | Use pgx/v5/pgconn | ✅ |
| 3 | RBAC missing AuthenticatedUser context | High (role management broken) | Bridge in middleware | ✅ |

---

## Remaining Known Limitations

1. **Docker build requires network access**: The multi-stage Dockerfile runs `pnpm install` during the frontend build stage. If the build environment cannot reach `registry.npmjs.org`, the build fails. This is expected Docker behaviour, not a bug.

2. **OIDC login requires external provider**: The `/api/v1/auth/login` endpoint returns 500 in development without a real Zitadel instance. This is expected — the BOOTSTRAP.md correctly documents using `DEV_TOKEN_SECRET` for local dev.

3. **`internal/auth/middleware.go` is dead code**: As documented in REPOSITORY_OVERVIEW.md, this legacy file is superseded by `internal/modules/rbac/middleware.go`. Can be safely deleted.

4. **AuthenticatedUser still in use**: The fix for Bug #3 is a compatibility bridge. The proper fix (documented as tech debt) is to update `rbac/handler.go` to use `MembershipContext` directly.

5. **`github.com/jackc/pgconn` v1 still in go.mod**: After fix #2, this dependency may be removable if no other code imports it. Not removed to minimize change scope.

---

## Verification Evidence

### Bootstrap
- `.env.example` contains all required variables with development defaults
- `docker compose up -d` starts postgres (healthy), openobserve, app (healthy after migrations), traefik
- 16 migrations apply on first start, skip on subsequent starts
- Frontend embedded and served at `/` (HTML response confirmed)

### API Endpoints Verified Against API_REFERENCE.md
- `GET /health` → `ok` (plain text) ✓
- `GET /health/live` → `{"status":"ok"}` ✓
- `GET /health/ready` → `{"status":"ready"}` ✓
- `GET /api/v1/ping` → `pong` ✓
- `GET /api/v1/token` → JWT token (dev only) ✓
- `GET /api/v1/auth/me` (no cookie) → 401 ✓
- `GET /api/v1/auth/me` (valid cookie) → user data ✓
- `POST /api/v1/auth/logout` → clears cookie ✓
- `POST /api/v1/organizations` → 201 with org data ✓
- `POST /api/v1/organizations` (duplicate) → 409 ✓ (after fix)
- `POST /api/v1/onboarding` → 201, creates org+user+roles ✓
- `GET /api/v1/memberships` → membership list ✓
- `GET /api/v1/roles` → 4 default roles with permissions ✓
- `POST /api/v1/roles` → 201, custom role created ✓ (after fix)
- `GET /api/v1/permissions` → 7 permissions ✓
- `GET /api/v1/billing` → subscription data ✓
- `POST /api/v1/billing` → 201, subscription created ✓
- `GET /api/v1/billing/subscription` → plan/status ✓
- `GET /api/v1/billing/usage` → usage counters ✓
- `GET /api/v1/billing/entitlements` → entitlement array ✓
- `POST /api/v1/invitations` → 201 with token ✓
- `GET /api/v1/audit` → audit log entries ✓ (after fix)
- `GET /api/v1/organizations/dashboard` → aggregated data ✓
- `GET /api/v1/organizations/summary` → org+plan summary ✓
- `GET /api/v1/organizations/usage-summary` → usage metrics ✓
- `GET /metrics` (no token) → 401 ✓
- `GET /metrics` (bearer token) → 200, Prometheus format ✓

### Security
- Session cookie: `HttpOnly=true`, `SameSite=Lax`, `Secure=false` (dev), `Path=/` ✓
- AES-GCM encryption with SHA256-derived key from SESSION_SECRET ✓
- Expired sessions rejected ✓
- Invalid cookies return 401 ✓
- Protected endpoints return 401 without session ✓
- Permission-gated endpoints return 403 without required permission ✓
- Metrics endpoint requires bearer token ✓
- Containers: `no-new-privileges`, `read_only` (app), non-root (UID 10001) ✓

### Persistence
- Postgres data persists across `docker compose restart` ✓
- App restarts cleanly without re-running migrations ✓
- Session cookies remain valid across app restarts (same SECRET) ✓
- Named volumes (postgres_data, openobserve_data, letsencrypt_data) declared ✓

---

## Overall Production Readiness Score

**8.5 / 10**

**Strengths**:
- Exceptional documentation quality (10 comprehensive documents)
- Clean architecture with enforced module boundaries
- Defense-in-depth security (RLS + application-level filtering)
- Single-command bootstrap experience
- All tests pass, code compiles cleanly
- Proper separation of concerns throughout

**Deductions**:
- -0.5: Three functional bugs found (now fixed)
- -0.5: Legacy compatibility layer (AuthenticatedUser) creates tech debt
- -0.5: No integration test coverage for the critical flows that were broken (audit list, org duplicate, role creation)

**Recommendation**: Ready for staging deployment after verifying the OIDC integration with a real Zitadel instance and adding integration tests for the three fixed flows.
