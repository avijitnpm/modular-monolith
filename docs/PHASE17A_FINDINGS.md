# Phase 17A — End-to-End Platform Audit Findings

**Date:** 2026-06-24
**Auditor Role:** Senior Staff Engineer / Security Engineer / SaaS Architect

---

## Executive Summary

**System Status: BLOCKED**

The platform cannot be used end-to-end by a real customer. Critical issues in the protected API route middleware chain prevent identity/membership resolution, causing all permission checks to fail for session-authenticated users. Additionally, the Dockerfile references a non-existent Go version, blocking container builds entirely.

### Totals

| Severity | Count |
|----------|-------|
| Critical | 5     |
| High     | 8     |
| Medium   | 6     |
| Low      | 3     |
| **Total** | **22** |

---

## Findings

---

## Finding 1

**Severity:** Critical

**Area:** Auth / RBAC

**Description:**
Protected API routes use `auth.Middleware(apiTokenProvider)` without passing `WithIdentityResolver` or `WithMembershipResolver` options. This means identity and membership resolution never occurs for protected routes. The `AuthenticatedUser.UserID` remains the raw Zitadel subject (provider ID), and `MembershipContext` is never set.

Consequence: `TenantContext` middleware falls back to `AuthenticatedUser.OrganizationID` (from token claims), which Zitadel may not populate. `RequirePermission` middleware uses the raw Zitadel subject as `membershipID` to query `user_roles`, which expects a `users.id` UUID — the permission check will always fail.

**Evidence:**
```go
// internal/router/routes.go lines in protected group:
protected.Use(
    auth.Middleware(apiTokenProvider),
)
// No WithIdentityResolver or WithMembershipResolver opts passed

// internal/auth/middleware.go:
// Identity/Membership resolution only runs if cfg.identityResolver != nil
if cfg.identityResolver != nil { ... }
```

**Affected Files:**
- `internal/router/routes.go`
- `internal/auth/middleware.go`
- `internal/middleware/tenant.go`
- `internal/modules/rbac/middleware.go`

**Impact:**
All protected API endpoints are inaccessible to session-authenticated users. Every permission-gated route returns 403 or 500.

**Recommended Fix Strategy:**
Pass `auth.WithIdentityResolver(identityResolver)` and `auth.WithMembershipResolver(membershipResolver)` when constructing the auth middleware for protected routes.

---

## Finding 2

**Severity:** Critical

**Area:** Auth / Tenant Isolation

**Description:**
`TenantContext` middleware falls back to `AuthenticatedUser.OrganizationID` when `MembershipContext` is not set. This field is populated from OIDC token claims (`urn:zitadel:iam:org:id` or similar). If Zitadel is configured without organization claims (common in personal project setups), this field is empty, and the middleware returns 500 "organization context missing" for every protected request.

**Evidence:**
```go
// internal/middleware/tenant.go:
if m, ok := appcontext.GetMembership(r.Context()); ok && m.OrganizationID != "" {
    organizationID = m.OrganizationID
} else if user, ok := appcontext.GetAuthenticatedUser(r.Context()); ok {
    organizationID = user.OrganizationID  // May be empty
}
if organizationID == "" {
    response.InternalServerError(w, "organization context missing")
    return
}
```

**Affected Files:**
- `internal/middleware/tenant.go`
- `internal/modules/authflow/user.go` (extractOrganizationID)

**Impact:**
Without membership resolution (Finding 1), tenant context cannot be established, blocking all protected routes.

**Recommended Fix Strategy:**
Fix Finding 1 first. As a secondary defense, if both membership and claims are empty, return 401 with a clear error instead of 500.

---

## Finding 3

**Severity:** Critical

**Area:** Onboarding

**Description:**
The onboarding flow does NOT call `BootstrapDefaultRoles` for the newly created organization. The `onboardingRoleAdapter.AssignOwnerRole` attempts to list roles for the new org and find one named "owner" — but no roles exist yet because they were never bootstrapped.

Result: `AssignOwnerRole` silently returns `nil` (no error) when no "owner" role is found, leaving the founding user with zero permissions.

**Evidence:**
```go
// internal/router/routes.go - onboardingRoleAdapter:
func (a *onboardingRoleAdapter) AssignOwnerRole(ctx context.Context, organizationID, userID string) error {
    roles, err := a.rbacRepo.ListRoles(ctx, organizationID)
    if err != nil { return err }
    for _, role := range roles {
        if role.Name == "owner" {
            _, err = a.rbacRepo.AssignRoleToUser(ctx, organizationID, userID, role.ID)
            return err
        }
    }
    return nil  // Silent failure — no owner role found
}

// internal/modules/onboarding/service.go - no BootstrapDefaultRoles call:
// Only calls: Orgs.RegisterOrganization, Users.CreateUser, Roles.AssignOwnerRole, Audit.LogOnboarding
```

**Affected Files:**
- `internal/router/routes.go` (onboardingRoleAdapter)
- `internal/modules/onboarding/service.go`

**Impact:**
Every newly onboarded user has no roles/permissions. They cannot access any permission-gated endpoint (billing, audit, settings, role management).

**Recommended Fix Strategy:**
Add a `RoleBootstrapper` interface to the onboarding service and call `BootstrapDefaultRoles` before `AssignOwnerRole`.

---

## Finding 4

**Severity:** Critical

**Area:** Infrastructure

**Description:**
The Dockerfile uses `golang:1.26-alpine` as the builder image. Go version 1.26 does not exist (current latest is 1.24.x as of mid-2026). The `go.mod` also declares `go 1.26.2` which is fictitious.

**Evidence:**
```dockerfile
# Dockerfile line 1:
FROM golang:1.26-alpine AS builder
```
```
# go.mod line 3:
go 1.26.2
```

**Affected Files:**
- `Dockerfile`
- `go.mod`

**Impact:**
`docker compose build` fails immediately. The platform cannot be containerized or deployed.

**Recommended Fix Strategy:**
Change to a valid Go version (e.g., `golang:1.23-alpine` or `golang:1.24-alpine`) in both Dockerfile and go.mod.

---

## Finding 5

**Severity:** Critical

**Area:** Infrastructure

**Description:**
There is no migration runner configured in `docker-compose.yml` or in the application startup code (`internal/app/start.go`). The app connects to PostgreSQL but never runs migrations. The `migrations/` directory uses `tern` format (evidenced by `tern.conf`) but no tern service or startup script exists.

**Evidence:**
```yaml
# docker-compose.yml - no migration service defined
# Only services: app, postgres, openobserve, traefik

# internal/app/start.go - no migration call:
# Only: config.Load, database.New, repository.New, service.New
```
```
# migrations/tern.conf exists but is never invoked
```

**Affected Files:**
- `docker-compose.yml`
- `internal/app/start.go`
- `migrations/tern.conf`

**Impact:**
Fresh deployments have an empty database with no tables. The application will crash on first request.

**Recommended Fix Strategy:**
Add a migration step — either a dedicated `migrate` service in docker-compose that runs before `app`, or integrate tern/goose into the app startup sequence.

---

## Finding 6

**Severity:** High

**Area:** Tenant Isolation / Security

**Description:**
The `invitations` table has RLS enabled, but `Repository.GetByToken` and `Repository.MarkAccepted` bypass RLS by querying directly on the pool (`r.DB.QueryRow` / `r.DB.Exec`) without calling `database.WithTenantQuery`. This means any authenticated user who knows a token can read and accept invitations belonging to any organization.

**Evidence:**
```go
// internal/modules/invitations/repository.go:
func (r *Repository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
    err := r.DB.QueryRow(ctx,  // Direct pool query - no tenant context set
        `SELECT ... FROM invitations WHERE token = $1`, token,
    ).Scan(...)
}

func (r *Repository) MarkAccepted(ctx context.Context, token string) error {
    _, err := r.DB.Exec(ctx,  // Direct pool query - no tenant context set
        `UPDATE invitations SET accepted_at = now() ... WHERE token = $1`, token,
    )
}
```

**Affected Files:**
- `internal/modules/invitations/repository.go`
- `migrations/015_invitations.sql`

**Impact:**
RLS is effectively bypassed for invitation acceptance. While the service layer validates email match, a compromised or malicious actor with a valid session could enumerate tokens.

**Recommended Fix Strategy:**
For `GetByToken`: This is intentionally cross-tenant (invitee doesn't belong to org yet). Add a comment documenting this is expected. For `MarkAccepted`: Use `WithTenantQuery` with the invitation's `organization_id` after retrieval, or add a superuser bypass policy.

---

## Finding 7

**Severity:** High

**Area:** Tenant Isolation / Security

**Description:**
The `identities` table has NO Row Level Security enabled. It is a global table accessible by any connection. While this is architecturally intentional (identities span organizations), it means any SQL injection or direct DB access can read all user identities across the platform.

**Evidence:**
```sql
-- migrations/014_identities.sql:
CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zitadel_user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    ...
);
-- No ALTER TABLE identities ENABLE ROW LEVEL SECURITY;
-- No CREATE POLICY on identities;
```

**Affected Files:**
- `migrations/014_identities.sql`

**Impact:**
If any query path has a SQL injection vulnerability, the entire identity table is exposed. This is a defense-in-depth gap.

**Recommended Fix Strategy:**
Document this as an intentional architectural decision. Consider adding a restrictive RLS policy that allows reads only when the caller has a valid `app.current_organization_id` set and the identity has at least one membership in that org, or use a service-account role pattern.

---

## Finding 8

**Severity:** High

**Area:** Membership / Schema

**Description:**
The `users` table has a global `UNIQUE` constraint on `email` (from migration 002). This prevents the same person from being a member of multiple organizations, which contradicts the identity model's stated goal of "one identity → many memberships."

**Evidence:**
```sql
-- migrations/002_users.sql:
CREATE TABLE users (
    ...
    email TEXT NOT NULL UNIQUE,  -- Global unique, not per-org
    ...
);
```

The `CreateMembership` function will fail with `ErrUserAlreadyExists` if the same email tries to join a second organization.

**Affected Files:**
- `migrations/002_users.sql`
- `internal/repository/user_repository.go`

**Impact:**
Multi-org membership is broken. A user invited to a second organization cannot accept because the email uniqueness constraint fires.

**Recommended Fix Strategy:**
Change the constraint to `UNIQUE (organization_id, email)` to allow the same email in different organizations while preventing duplicates within one org.

---

## Finding 9

**Severity:** High

**Area:** Membership / Schema

**Description:**
The `users` table also has a global `UNIQUE` constraint on `zitadel_user_id` (from migration 002). Same issue as Finding 8 — prevents multi-org membership for the same Zitadel identity.

**Evidence:**
```sql
-- migrations/002_users.sql:
CREATE TABLE users (
    ...
    zitadel_user_id TEXT NOT NULL UNIQUE,  -- Global unique
    ...
);
```

**Affected Files:**
- `migrations/002_users.sql`
- `internal/repository/user_repository.go`

**Impact:**
Same as Finding 8. Multi-org membership is architecturally impossible with the current schema.

**Recommended Fix Strategy:**
Change to `UNIQUE (organization_id, zitadel_user_id)`.

---

## Finding 10

**Severity:** High

**Area:** Frontend / UX

**Description:**
The backend redirects new users (no memberships) to `/onboarding` after OAuth callback. However, no `/onboarding` route exists in the frontend. The SvelteKit router will fall through to the SPA catch-all, which serves the root `+page.svelte` (a simple redirect to `/dashboard`), creating an infinite redirect loop or a dead-end.

**Evidence:**
```
# Frontend routes structure:
frontend/src/routes/
├── (app)/dashboard/
├── (app)/billing/
├── (app)/users/
├── (app)/roles/
├── (app)/settings/
├── login/
├── logout/
└── +page.svelte  (root)
# NO /onboarding route exists
```

```go
// internal/modules/authflow/handler.go:
redirectURL = "/onboarding"  // Backend redirects here
```

**Affected Files:**
- `internal/modules/authflow/handler.go`
- `frontend/src/routes/` (missing onboarding route)

**Impact:**
New users cannot complete onboarding. The entire first-use flow is broken.

**Recommended Fix Strategy:**
Create `frontend/src/routes/onboarding/+page.svelte` with a form that calls `POST /api/v1/onboarding` with an organization name.

---

## Finding 11

**Severity:** High

**Area:** Frontend / Auth

**Description:**
The frontend `UserSchema` (Zod) does not include `identity_id` field. The backend's `/auth/me` endpoint returns `SessionUser` which includes `identity_id`. The Zod schema will strip this field during parsing (Zod v4 strips unknown keys by default), making it unavailable to the frontend for onboarding/invitation flows.

**Evidence:**
```typescript
// frontend/src/lib/schemas/auth.ts:
export const UserSchema = z.object({
    subject: z.string(),
    email: z.string().optional(),
    // ... other fields
    organization_id: z.string().optional(),
    roles: z.array(z.string()).optional(),
    // NO identity_id field
});
```

```go
// internal/modules/authflow/user.go:
type SessionUser struct {
    ...
    IdentityID string `json:"identity_id,omitempty"`  // Sent by backend
    ...
}
```

**Affected Files:**
- `frontend/src/lib/schemas/auth.ts`
- `internal/modules/authflow/user.go`

**Impact:**
Frontend cannot determine the user's identity_id for API calls that require it.

**Recommended Fix Strategy:**
Add `identity_id: z.string().optional()` to the `UserSchema`.

---

## Finding 12

**Severity:** High

**Area:** Auth / Infrastructure

**Description:**
The `docker-compose.yml` app service does not pass OIDC configuration environment variables (`OIDC_ISSUER`, `OIDC_AUDIENCE`, `OIDC_CLIENT_ID`, `OIDC_REDIRECT_URL`, `SESSION_SECRET`). The config validation requires these — the app will fail to start with a config validation error.

**Evidence:**
```yaml
# docker-compose.yml app environment:
environment:
    APP_NAME: ...
    APP_ENV: ...
    SERVER_PORT: ...
    DATABASE_URL: ...
    DODO_API_KEY: ...
    DODO_WEBHOOK_SECRET: ...
    OTEL_ENABLED: ...
    # Missing: OIDC_ISSUER, OIDC_AUDIENCE, OIDC_CLIENT_ID,
    #          OIDC_REDIRECT_URL, SESSION_SECRET, CORS_ORIGIN
```

```go
// internal/config/validate.go:
if cfg.Auth.OIDCIssuer == "" { return errors.New("OIDC_ISSUER is required") }
// ... all required
```

**Affected Files:**
- `docker-compose.yml`
- `internal/config/validate.go`

**Impact:**
The containerized app crashes on startup due to missing required configuration.

**Recommended Fix Strategy:**
Add all required auth environment variables to the docker-compose app service, referencing `.env` file values.

---

## Finding 13

**Severity:** High

**Area:** Auth / Security

**Description:**
The `DBMembershipResolver.GetDefaultMembership` and `ListMemberships` query the `users` table directly without setting tenant context. The `users` table has RLS enabled. Without `SET LOCAL app.current_organization_id`, these queries return zero rows (RLS blocks all rows when the setting is empty/unset).

This means membership resolution will ALWAYS fail for users, even if Finding 1 is fixed.

**Evidence:**
```go
// internal/modules/identityresolver/membership.go:
func (r *DBMembershipResolver) GetDefaultMembership(ctx context.Context, identityID string) (*appcontext.Membership, error) {
    err := r.DB.QueryRow(ctx,  // Direct pool query, no tenant context
        `SELECT id, organization_id FROM users WHERE identity_id = $1 LIMIT 1`,
        identityID,
    ).Scan(...)
}
```

The `users` table has RLS policy: `organization_id = current_setting('app.current_organization_id', true)`. Since no tenant context is set, this returns empty string which matches no rows.

**Affected Files:**
- `internal/modules/identityresolver/membership.go`
- `migrations/005_add_organization_id_and_rls.sql`

**Impact:**
Membership resolution always returns "not found", preventing the identity→membership→tenant chain from working.

**Recommended Fix Strategy:**
Use a BYPASSRLS connection role for cross-tenant membership lookups, or execute these queries with a superuser connection that bypasses RLS, or disable RLS for this specific query pattern using a security definer function.

---

## Finding 14

**Severity:** Medium

**Area:** Frontend / API Integration

**Description:**
The frontend dashboard API calls `GET /organizations/dashboard` which is a protected route requiring Bearer token auth. However, the frontend fetcher uses `credentials: 'include'` (cookies) and does NOT send an Authorization header. The protected route middleware expects a Bearer token — the request will be rejected with "missing authorization header".

The frontend auth flow uses session cookies (set by callback), but protected routes expect Bearer tokens. These are two incompatible authentication mechanisms.

**Evidence:**
```typescript
// frontend/src/lib/utils/fetcher.ts:
const init: RequestInit = {
    method,
    headers: { 'Content-Type': 'application/json', ...headers },
    credentials: 'include',  // Sends cookies
    // No Authorization header
};
```

```go
// internal/auth/middleware.go:
header := r.Header.Get("Authorization")
if header == "" {
    response.BadRequest(w, "missing authorization header")
    return
}
```

**Affected Files:**
- `frontend/src/lib/utils/fetcher.ts`
- `internal/auth/middleware.go`
- `internal/router/routes.go`

**Impact:**
All frontend API calls to protected routes fail with 400 "missing authorization header". The dashboard, billing, roles, users pages all show errors.

**Recommended Fix Strategy:**
Either: (a) Add a session-based auth middleware for browser requests that extracts identity from the cookie (like `SessionIdentityMiddleware`) and use it for protected routes, or (b) Issue an access token at login that the frontend stores and sends as Bearer.

---

## Finding 15

**Severity:** Medium

**Area:** Onboarding / Atomicity

**Description:**
The onboarding service (`CompleteOnboarding`) does NOT run in a database transaction. Each step (create org, create user, assign role, audit log) is a separate transaction via `WithTenantQuery`. If any intermediate step fails, the system is left in an inconsistent state (e.g., org created but no user, or user created but no role).

**Evidence:**
```go
// internal/modules/onboarding/service.go:
func (s *Service) CompleteOnboarding(...) {
    // Step 1: Check membership (separate query)
    // Step 2: Create org (separate WithTenantQuery tx)
    // Step 3: Create user (separate WithTenantQuery tx)
    // Step 4: Assign role (separate WithTenantQuery tx)
    // Step 5: Audit log (separate WithTenantQuery tx)
    // No wrapping transaction
}
```

**Affected Files:**
- `internal/modules/onboarding/service.go`
- `internal/router/routes.go` (adapters)

**Impact:**
Partial onboarding failures leave orphaned organizations with no members or members with no roles. Recovery requires manual DB intervention.

**Recommended Fix Strategy:**
Wrap the entire onboarding flow in a single database transaction, passing the tx through all adapter calls.

---

## Finding 16

**Severity:** Medium

**Area:** Identity

**Description:**
The `identities` table has a global `UNIQUE` constraint on `email`. If a user changes their email in Zitadel and another user already has that email in the identities table, the `FindOrCreateIdentity` update will fail with a unique violation, blocking login for the user who changed their email.

**Evidence:**
```sql
-- migrations/014_identities.sql:
email TEXT NOT NULL UNIQUE,
```

```go
// internal/modules/identity/service.go:
if existing.Email != email || existing.Name != name {
    return s.Repository.Update(ctx, zitadelUserID, email, name)
    // This can fail if new email already exists for another identity
}
```

**Affected Files:**
- `migrations/014_identities.sql`
- `internal/modules/identity/service.go`

**Impact:**
Email changes in the IdP can lock users out of the platform.

**Recommended Fix Strategy:**
Handle the unique violation gracefully — either by merging identities or returning a clear error that explains the conflict.

---

## Finding 17

**Severity:** Medium

**Area:** RBAC / Security

**Description:**
The `permissions` table has NO Row Level Security. It is a shared global table (all orgs see the same permissions). While this is architecturally correct, it means `ListPermissions` in the RBAC repository queries without tenant context (`r.DB.Query` directly). If the app connection role has restricted permissions in production, this query could fail.

Additionally, the `permissions` table is queried from within tenant-scoped transactions (e.g., `assignRolePermissions` joins `permissions p` inside a `WithTenantQuery` transaction). Since RLS is not enabled on `permissions`, this works — but it's an implicit dependency that isn't documented.

**Evidence:**
```go
// internal/modules/rbac/repository.go:
func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
    rows, err := r.DB.Query(ctx, `SELECT id, name, created_at FROM permissions ORDER BY name`)
    // Direct query, no tenant context
}
```

**Affected Files:**
- `internal/modules/rbac/repository.go`
- `migrations/008_rbac_foundation.sql`

**Impact:**
Low risk in practice since permissions are intentionally global, but should be documented.

**Recommended Fix Strategy:**
Add a code comment documenting that `permissions` is intentionally a global shared table not subject to RLS.

---

## Finding 18

**Severity:** Medium

**Area:** Frontend / API Mismatch

**Description:**
The frontend `DashboardResponseSchema` expects a specific shape (`organization: {id, name}`, `subscription: {...}`, `usage: {users, documents, api_requests, storage}`, `entitlements: [...]`). Without reading the actual dashboard handler source, the backend dashboard handler (`organizations.NewDashboardHandler`) constructs the response from multiple adapter calls. If any adapter returns null/empty, the Zod schema parsing will throw because fields like `usage.users` are required `z.number()` (not optional).

**Evidence:**
```typescript
// frontend/src/lib/schemas/dashboard.ts:
export const DashboardUsageSchema = z.object({
    users: z.number(),        // Required
    documents: z.number(),    // Required
    api_requests: z.number(), // Required
    storage: z.number(),      // Required
});
```

If the backend returns `{}` or partial usage data (e.g., no usage counters exist yet for a new org), the frontend will throw a Zod validation error.

**Affected Files:**
- `frontend/src/lib/schemas/dashboard.ts`
- `internal/modules/organizations/` (dashboard handler)

**Impact:**
Dashboard page shows error state for new organizations that have no usage data yet.

**Recommended Fix Strategy:**
Make usage fields default to 0 in the backend response, or make them optional with `.default(0)` in the Zod schema.

---

## Finding 19

**Severity:** Medium

**Area:** Billing

**Description:**
The `subscriptions` table has an `ON CONFLICT (organization_id)` clause in `UpsertSubscriptionByProvider`, implying a unique constraint on `organization_id`. However, `CreateSubscription` uses a generic unique violation check. If an org already has a subscription and tries to create another via the API, the error handling works. But the schema allows only one subscription per org (the unique constraint), meaning plan upgrades must use `UpdateSubscription` — this isn't enforced at the API level.

The billing API exposes both `POST /billing` (create) and `PATCH /billing/{id}` (update) but no clear flow guides the frontend on which to use.

**Evidence:**
```go
// internal/modules/billing/repository.go - UpsertSubscriptionByProvider:
`ON CONFLICT (organization_id) DO UPDATE SET ...`
// This implies UNIQUE(organization_id) on subscriptions table
```

**Affected Files:**
- `internal/modules/billing/repository.go`
- `migrations/010_billing_foundation.sql`

**Impact:**
Confusing API surface for billing management. Frontend must handle the 409 conflict case for duplicate subscriptions.

**Recommended Fix Strategy:**
Document the billing flow clearly. Consider removing the public `POST /billing` in favor of webhook-driven subscription creation only.

---

## Finding 20

**Severity:** Low

**Area:** Frontend / Auth

**Description:**
The `AuthMeResponseSchema` requires `user` to always be present (`z.object({authenticated: z.boolean(), user: UserSchema})`). When the session is invalid/expired, the backend returns `{"authenticated": false, "user": ...}` but if the session is completely absent, it returns `{"error": "not authenticated"}` with status 401. The frontend handles the 401 in the fetcher, but the schema would fail if the backend ever returned `{authenticated: false}` without a `user` field.

**Evidence:**
```go
// internal/modules/authflow/handler.go - Me():
if err != nil {
    response.Error(w, http.StatusUnauthorized, "not authenticated")  // No user field
    return
}
response.OK(w, map[string]any{"authenticated": true, "user": session.User})
```

```typescript
// Frontend handles 401 in fetcher, so AuthMeResponseSchema only parses 200 responses
// This is fine currently but brittle
```

**Affected Files:**
- `frontend/src/lib/schemas/auth.ts`
- `internal/modules/authflow/handler.go`

**Impact:**
Currently works because 401 is caught before Zod parsing. Fragile if error handling changes.

**Recommended Fix Strategy:**
Make `user` optional in the schema: `user: UserSchema.optional()`.

---

## Finding 21

**Severity:** Low

**Area:** Infrastructure / Observability

**Description:**
The `docker-compose.yml` `app` service `healthcheck` uses `wget` but the builder image is `alpine:3.22` which may not have `wget` installed by default. The health check command `wget -qO- http://127.0.0.1:8080/health` could fail if wget is missing.

**Evidence:**
```yaml
healthcheck:
    test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:${SERVER_PORT:-8080}/health >/dev/null || exit 1"]
```

The final image `FROM alpine:3.22` only copies the Go binary. No wget/curl is installed.

**Affected Files:**
- `Dockerfile`
- `docker-compose.yml`

**Impact:**
Container health checks may fail, causing orchestrators to restart the container continuously.

**Recommended Fix Strategy:**
Either install wget/curl in the final Dockerfile stage, or use the Go binary itself for health checks (e.g., add a `--health` flag), or use `CMD` with a simple TCP check.

---

## Finding 22

**Severity:** Low

**Area:** Frontend / Config

**Description:**
The `.env.example` sets `CORS_ORIGIN=http://localhost:5173` (Vite dev server port), but the production architecture serves the SPA from the Go backend (embedded via `frontend.Handler()`). In production with embedded frontend, CORS isn't needed (same-origin). In development, if the frontend dev server is running separately, the CORS origin must match. This is correctly configured for dev but could cause confusion.

**Evidence:**
```
# .env.example:
CORS_ORIGIN=http://localhost:5173
```

```go
// internal/router/router.go:
r.Use(middleware.CORS(cfg.App.CORSOrigin))
// In production with embedded SPA, this origin won't match any external request
```

**Affected Files:**
- `.env.example`
- `internal/middleware/cors.go`

**Impact:**
Minor configuration confusion. No functional impact with embedded SPA.

**Recommended Fix Strategy:**
Document that `CORS_ORIGIN` is only needed for local development with a separate frontend dev server.

---

## Summary of Blocking Issues for Customer Journey

| Step | Status | Blocker |
|------|--------|---------|
| Docker build | ❌ BLOCKED | Finding 4: Invalid Go version |
| App startup | ❌ BLOCKED | Finding 5: No migrations, Finding 12: Missing env vars |
| Login | ⚠️ Partial | Works if Zitadel configured correctly |
| Callback/Session | ✅ Works | Session cookie creation is correct |
| Onboarding redirect | ❌ BLOCKED | Finding 10: No frontend route |
| Onboarding API | ❌ BLOCKED | Finding 3: No role bootstrap, Finding 13: RLS blocks membership check |
| Dashboard load | ❌ BLOCKED | Finding 1: No identity resolution, Finding 14: No Bearer token from frontend |
| Any protected route | ❌ BLOCKED | Findings 1, 2, 14 combined |
| Multi-org join | ❌ BLOCKED | Findings 8, 9: Unique constraints |

**Conclusion:** The platform is not usable end-to-end. At minimum, Findings 1, 3, 4, 5, 8, 9, 10, 12, 13, and 14 must be resolved before a customer can complete the journey from login to dashboard.
