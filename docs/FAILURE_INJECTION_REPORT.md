# Failure Injection & Resilience Validation Report

**Date**: 2026-06-27
**Environment**: Docker Compose (postgres:17.5-alpine, Go 1.26, local development stack)
**Methodology**: Manual failure injection against running system, code review, direct DB testing

---

## Executive Summary

The modular-monolith demonstrates **strong resilience** across most failure categories. Configuration validation is excellent, authentication boundaries are robust, database constraints are properly enforced, and the application handles infrastructure failures gracefully without panicking.

**Two significant findings require attention before production:**

1. **CRITICAL** — The application connects to PostgreSQL as the `postgres` superuser, which **bypasses Row Level Security (RLS)**. RLS policies exist and are correctly defined but provide zero protection when the connection role is a superuser. Tenant isolation relies entirely on application-layer `WithTenantQuery` correctness.

2. **HIGH** — The logging middleware does not capture HTTP response status codes, client IP addresses, or error details. An operator investigating a 3 AM incident cannot determine from logs alone what is failing, for whom, or why.

**Operational Readiness Score: 8/10**

The system is well-architected for resilience. Logging now captures full request context for incident response. The remaining gap is the PostgreSQL superuser connection which bypasses RLS — this is a deployment configuration issue, not an application defect.

---

## Failure Matrix

| # | Category | Test | Expected | Observed | Result |
|---|----------|------|----------|----------|--------|
| 1.1 | Infrastructure | Kill PostgreSQL | Readiness 503, liveness OK | Readiness 503, liveness OK, no panic | **PASS** |
| 1.2 | Infrastructure | Kill OpenObserve | App unaffected | App unaffected, OTEL non-blocking | **PASS** |
| 1.3 | Infrastructure | Invalid DATABASE_URL | Fail fast | Panic with "context deadline exceeded" within 5s | **PASS** |
| 2.1 | Restart | Postgres restart | Auto-reconnect | pgxpool reconnects transparently | **PASS** |
| 2.2 | Restart | App restart | Migrations idempotent | No migrations re-applied | **PASS** |
| 2.3 | Restart | Session survival | Cookies survive restart | Stateless encrypted cookies, survive | **PASS** |
| 3.1 | Config | Missing DATABASE_URL | Fast fail | `panic: DATABASE_URL is required` | **PASS** |
| 3.2 | Config | Missing SESSION_SECRET | Fast fail | `panic: SESSION_SECRET is required` | **PASS** |
| 3.3 | Config | Short SESSION_SECRET | Fast fail | `panic: SESSION_SECRET must be at least 32 characters` | **PASS** |
| 3.4 | Config | Placeholder secret in production | Fast fail | `panic: SESSION_SECRET must not use the default placeholder` | **PASS** |
| 3.5 | Config | DEV_TOKEN_SECRET in production | Fast fail | `panic: DEV_TOKEN_SECRET must not be set in production` | **PASS** |
| 3.6 | Config | sslmode=disable in production | Fast fail | `panic: DATABASE_URL must not use sslmode=disable` | **PASS** |
| 3.7 | Config | HTTP OIDC_REDIRECT_URL in production | Fast fail | `panic: OIDC_REDIRECT_URL must use https://` | **PASS** |
| 3.8 | Config | Missing OIDC_ISSUER | Fast fail | `panic: OIDC_ISSUER is required` | **PASS** |
| 4.1 | Auth | No session cookie | 401 | `{"error":"not authenticated"}` HTTP 401 | **PASS** |
| 4.2 | Auth | Invalid/corrupted cookie | 401 | `{"error":"not authenticated"}` HTTP 401 | **PASS** |
| 4.3 | Auth | Forged cookie (random bytes) | 401 | `{"error":"not authenticated"}` HTTP 401 | **PASS** |
| 4.4 | Auth | Memberships endpoint without auth | 401 | `{"error":"not authenticated"}` HTTP 401 | **PASS** |
| 4.5 | Auth | Information leakage | No internal details | Generic error messages only | **PASS** |
| 5.1 | DB Constraints | Duplicate organization (unique) | PG error | `duplicate key value violates unique constraint` | **PASS** |
| 5.2 | DB Constraints | Duplicate identity email | PG error | `duplicate key value violates unique constraint "identities_email_key"` | **PASS** |
| 5.3 | DB Constraints | FK violation (invalid identity_id) | PG error | `violates foreign key constraint "fk_users_identity_id"` | **PASS** |
| 5.4 | DB Constraints | Duplicate role assignment | PG unique constraint | `user_roles_organization_id_user_id_role_id_key` UNIQUE | **PASS** |
| 6.1 | Tenant Isolation | RLS for non-superuser | Tenant sees own data only | Verified: Tenant A sees only org-a data | **PASS** |
| 6.2 | Tenant Isolation | RLS for superuser (postgres) | Should enforce isolation | **RLS BYPASSED — superuser sees ALL data** | **FAIL** |
| 6.3 | Tenant Isolation | WithTenantQuery empty org_id | Reject | Returns `"organization ID must not be empty"` | **PASS** |
| 6.4 | Tenant Isolation | Middleware chain enforcement | 401/500 without context | SessionIdentity→ResolveMembership→TenantContext chain enforced | **PASS** |
| 7.1 | Resource Exhaustion | Body > 1MB | 413 | `{"error":"request body too large"}` HTTP 413 | **PASS** |
| 7.2 | Resource Exhaustion | Rate limit (10 req/min public) | 429 after limit | HTTP 429 after 10th request | **PASS** |
| 7.3 | Resource Exhaustion | 100 concurrent requests | No panic | All responses correct, no panic | **PASS** |
| 8.1 | Operator Experience | Status code in logs | Present | **FIXED** — status code now logged | **FIXED** |
| 8.2 | Operator Experience | Error on readiness failure | ERROR level log | **FIXED** — ERROR with dependency + error | **FIXED** |
| 8.3 | Operator Experience | Client IP in logs | Present | **FIXED** — client_ip now logged | **FIXED** |
| 8.4 | Operator Experience | Trace correlation | trace_id in logs | Present via OTEL context | **PASS** |
| 8.5 | Operator Experience | Structured JSON logging | JSON format | Confirmed | **PASS** |

---

## Detailed Findings

### 1. Infrastructure Failure

#### PostgreSQL Kill

**Procedure**: `docker compose stop postgres` while app running

**Observed**:
- `/health/live` → 200 `{"status":"ok"}` (correct — liveness should not depend on DB)
- `/health/ready` → 503 `{"status":"not_ready"}` (correct — readiness indicates dependency failure)
- Application continues serving requests (non-DB routes work)
- API routes requiring DB return 500 with safe error messages
- No panic, no goroutine leak

**Recovery**: After `docker compose start postgres`, readiness returns to 200 within seconds. pgxpool handles reconnection transparently.

#### OpenObserve Kill

**Procedure**: `docker compose stop openobserve` while app running

**Observed**: Zero impact. The OTLP exporter operates asynchronously. No logs, no errors, no degradation. Application remains fully functional.

#### Startup with Unreachable Database

**Observed**: Application panics within 5 seconds (context deadline exceeded). This is correct fail-fast behavior — the Docker `restart: unless-stopped` policy ensures the container retries.

---

### 2. Restart Behaviour

| Scenario | Result |
|----------|--------|
| Postgres restart | pgxpool auto-reconnects; no intervention needed |
| App restart (migrations already applied) | Zero migrations re-applied; idempotent startup |
| Session persistence | Encrypted cookies are stateless; survive any restart |
| Data persistence | PostgreSQL volume-mounted; data survives full stack restart |

---

### 3. Configuration Failure

All configuration validation occurs at startup via `config.validate()`. Every missing or invalid configuration causes an immediate panic with a clear, actionable error message.

**Production-mode checks verified**:
- Placeholder SESSION_SECRET rejected
- DEV_TOKEN_SECRET must not be set
- OIDC_REDIRECT_URL must use HTTPS
- CORS_ORIGIN must use HTTPS
- DATABASE_URL must not use sslmode=disable
- OTEL insecure must be false
- METRICS_TOKEN required
- DODO_BASE_URL required

**Missing .env**: App logs `.env file not found, using system env` and proceeds with environment variables only. This is correct behavior for container deployments.

---

### 4. Authentication Failure

The `SessionIdentityMiddleware` correctly rejects all unauthenticated access:

- No cookie → 401
- Garbage cookie → 401
- Random base64 (forged) → 401
- Empty identity → 401

**No information leakage**: Error responses contain only `{"error":"not authenticated"}` with no internal details, stack traces, or implementation hints.

---

### 5. Database Constraints

All critical uniqueness and referential integrity constraints are enforced at the database level:

| Constraint | Enforcement |
|-----------|-------------|
| `organizations.zitadel_org_id` | UNIQUE |
| `identities.email` | UNIQUE |
| `identities.zitadel_user_id` | UNIQUE |
| `users.email` | UNIQUE |
| `users.zitadel_user_id` | UNIQUE |
| `users.identity_id → identities.id` | FK (ON DELETE RESTRICT) |
| `user_roles(org_id, user_id, role_id)` | UNIQUE (prevents duplicate assignments) |
| `user_roles.user_id → users.id` | FK (ON DELETE CASCADE) |
| `user_roles(org_id, role_id) → roles(org_id, id)` | FK (ON DELETE CASCADE) |

PostgreSQL returns proper constraint violation errors. The application code surfaces these as appropriate HTTP error responses.

---

### 6. Tenant Isolation

#### CRITICAL: Superuser Bypasses RLS

**Root Cause**: The `DATABASE_URL` in Docker Compose connects as the `postgres` superuser:
```
postgres://postgres:postgres@postgres:5432/app_db?sslmode=disable
```

PostgreSQL by design does **not enforce RLS policies against superusers or table owners**. This means:

1. If any code path accidentally queries without `WithTenantQuery`, there is **no database-level safety net**
2. If `WithTenantQuery` receives an incorrect org_id due to a bug, there is **no second line of defense**
3. Direct pool queries (bypassing `WithTenantQuery`) return **all tenant data**

**Verified**: When querying as a non-superuser with `set_config('app.current_organization_id', 'org-a', true)`, RLS correctly returns only org-a rows. The policies are correctly defined — they simply don't apply to the current connection role.

#### Application-Layer Defenses (Working)

The middleware chain provides defense-in-depth:
1. `SessionIdentityMiddleware` — rejects unauthenticated requests (401)
2. `ResolveMembershipMiddleware` — resolves identity → org membership from DB
3. `TenantContext` — extracts org_id; returns 500 if missing
4. `WithTenantQuery` — rejects empty org_id at repository layer

These defenses are correctly implemented, but they represent a **single point of failure** without RLS as a backup.

---

### 7. Resource Exhaustion

| Control | Limit | Enforcement |
|---------|-------|-------------|
| Request body size | 1 MB | `middleware.BodyLimit(1 << 20)` — HTTP 413 |
| Public rate limit | 10 req/min per IP | `middleware.PublicRateLimit()` — HTTP 429 |
| Authenticated rate limit | 60 req/min per user | `middleware.AuthenticatedRateLimit()` — HTTP 429 |
| Webhook rate limit | 120 req/min per IP | `middleware.WebhookRateLimit()` — HTTP 429 |
| Container memory | 512 MB (app) | Docker Compose deploy.resources.limits |
| Container CPU | 1.0 CPU (app) | Docker Compose deploy.resources.limits |
| Concurrent requests | Tested 100 parallel | No panic, no degraded responses |

---

### 8. Operator Experience

#### Issues Identified

**8.1 — Missing HTTP status code in access logs**

Current log output:
```json
{"level":"INFO","msg":"http request","method":"GET","path":"/health/ready","duration":"884.304µs"}
```

An operator cannot distinguish 200 from 503 from 500 without examining the response. Standard access logging includes status codes.

**8.2 — No error-level logging when readiness check fails**

When PostgreSQL is down, the readiness endpoint returns 503 but logs at INFO level with no error context. An operator scanning for `level=ERROR` would find nothing.

**8.3 — No client IP in access logs**

The logging middleware captures method, path, and duration but not the client's remote address. This makes it impossible to correlate with rate limiting, identify abusive clients, or investigate security incidents.

#### Working Well

- Structured JSON logging (production-ready for log aggregation)
- trace_id and span_id propagated via OpenTelemetry context
- PII redaction in logger (emails, tokens)
- Recovery middleware logs panics with full stack trace at ERROR level
- Startup logs clearly indicate service name, port, OTEL endpoint

---

## Recovery Procedures

### PostgreSQL Unavailable

1. Check `docker compose ps` — is postgres healthy?
2. Check `docker compose logs postgres --tail=50`
3. If crashed: `docker compose restart postgres` (auto-reconnection)
4. If data corruption: restore from backup, restart

### Application Crash Loop

1. Check `docker compose logs app --tail=100` — look for panic messages
2. Verify `.env` / secrets are present and valid
3. Verify postgres is healthy: `docker compose exec postgres pg_isready`
4. If config issue: fix env vars, `docker compose up -d app`

### OpenObserve Down

No action required. Application continues without observability. Restart when convenient: `docker compose restart openobserve`

---

## Known Remaining Risks

| Risk | Severity | Mitigation |
|------|----------|-----------|
| Superuser connection bypasses RLS | **CRITICAL** | Create dedicated `app` role with `NOLOGIN` superuser; use `app_user` role for application connections |
| No status code in access logs | HIGH | Wrap `http.ResponseWriter` to capture status; log at appropriate level |
| No error log on readiness failure | MEDIUM | Add `logger.Error()` in health handler when ping fails |
| No client IP in logs | MEDIUM | Add `r.RemoteAddr` or `X-Forwarded-For` to log attributes |
| Recovery middleware redacts panic error | LOW | Intentional (prevents info leakage), but may hinder debugging in rare cases |
| OpenObserve has no healthcheck | LOW | Distroless image has no shell; OTEL exporter handles gracefully |

---

## Recommendations (Operational Only)

These recommendations address **behaviour that violates documented expectations** (RLS as tenant isolation) or would prevent an operator from diagnosing production issues. No architectural changes proposed.

### 1. Create a dedicated PostgreSQL application user

```sql
CREATE ROLE app_user LOGIN PASSWORD '<generated>';
GRANT CONNECT ON DATABASE app_db TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
```

Update `DATABASE_URL` to use `app_user` instead of `postgres`. This enables RLS enforcement as documented in ARCHITECTURE.md.

### 2. ~~Add HTTP status code to access logs~~ ✅ Fixed in Phase 21B

The logging middleware now captures response status code, client IP, user agent, request_id, identity, and org_id using an `http.ResponseWriter` wrapper.

### 3. ~~Log readiness check failures at ERROR level~~ ✅ Fixed in Phase 21B

The health handler now emits an ERROR log with dependency name and error message when `db.Ping()` fails.

---

## Test Environment Cleanup

All test data created during this validation was removed. The Docker Compose stack was left in a running state with postgres and openobserve healthy.

---

## Fixed In Phase 21B

| Finding | Fix | Files Modified |
|---------|-----|---------------|
| 8.1 — No HTTP status code in access logs | Wrapped `http.ResponseWriter` with `loggingWriter` to capture status code | `internal/middleware/logging.go` |
| 8.2 — No ERROR log on readiness failure | Health handler now emits `level=ERROR` with dependency name and error | `internal/modules/health/handler.go` |
| 8.3 — No client IP in access logs | Logging middleware extracts client IP from X-Forwarded-For or RemoteAddr | `internal/middleware/logging.go` |

**Additional fields now logged per request**: `status`, `client_ip`, `user_agent`, `request_id`, `identity` (when authenticated), `org_id` (when in tenant context).

**Before** (Phase 21A):
```
level=INFO msg="http request" method=GET path=/health/ready duration=884.304µs
```

**After** (Phase 21B):
```
level=INFO msg="http request" method=GET path=/health/ready status=200 duration=346µs client_ip=::1 user_agent=curl/8.20.0 request_id=0f23507e-eed5-4dee-8240-360556d79eb0
```

**Readiness failure** (new ERROR log):
```
level=ERROR msg="readiness check failed" dependency=postgres error="failed to connect to `user=postgres database=app_db`: dial tcp 127.0.0.1:5432: connect: connection refused"
```

---

## Conclusion

The system demonstrates strong engineering discipline in failure handling. Configuration validation is comprehensive, authentication boundaries are correctly enforced, the application never panics under normal failure conditions, and recovery is automatic.

Phase 21B resolved all operator experience gaps: access logs now include response status codes, client IPs, user agents, and request context. Readiness failures emit ERROR-level logs with dependency identification and error details. An operator can now diagnose failures from logs alone.

The remaining RLS superuser bypass is a deployment configuration concern (use a non-superuser `DATABASE_URL` in production) documented in the Recommendations section above. It does not require application code changes.
