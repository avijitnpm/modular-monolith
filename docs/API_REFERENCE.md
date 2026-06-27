# API Reference

Complete HTTP API documentation for the modular-monolith platform.

---

## Conventions

### Base URL

All API endpoints are prefixed with `/api/v1` unless otherwise noted. Health endpoints are at the root.

### Authentication

Authentication uses encrypted session cookies (`mm_session`). The cookie is:
- AES-GCM encrypted (derived from `SESSION_SECRET`)
- `HttpOnly` — not accessible from JavaScript
- `SameSite=Lax`
- `Secure` in production (HTTPS only)
- Automatically set after successful OIDC callback

No `Authorization` header is needed — the browser sends the cookie automatically.

### Response Format

**Success responses** (via `response.OK` / `response.Created`):
```json
{"data": <payload>}
```

**Error responses**:
```json
{"error": "human-readable message"}
```

**Validation error responses** (400):
```json
{"error": "validation failed", "validation_errors": [{"field": "email", "message": "is required"}]}
```

### Pagination

List endpoints accept query parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 50 | Max items to return |
| `offset` | int | 0 | Number of items to skip |

### Rate Limiting

| Scope | Applied To |
|-------|-----------|
| Public | `/auth/login`, `/auth/callback`, `/organizations` (POST), `/onboarding`, `/invitations/accept` |
| Webhook | `/billing/webhook` |
| Authenticated write | `/roles` (POST), `/billing/checkout`, `/users/{id}/roles` |

Rate-limited endpoints return `429 Too Many Requests` when exceeded.

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad request / validation failed |
| 401 | Not authenticated |
| 403 | Permission denied |
| 404 | Resource not found |
| 409 | Conflict (duplicate resource) |
| 413 | Request body too large |
| 429 | Rate limited |
| 500 | Internal server error |
| 503 | Service unavailable |

---

## Health

### GET /health

Basic health check. Returns `ok` as plain text.

```bash
curl http://localhost:8080/health
```

### GET /health/live

Liveness probe. Application is running.

```bash
curl http://localhost:8080/health/live
```

**Response** `200`:
```json
{"status": "ok"}
```

### GET /health/ready

Readiness probe. Application can serve traffic (database connected).

```bash
curl http://localhost:8080/health/ready
```

**Response** `200`:
```json
{"status": "ready"}
```

**Response** `503`:
```json
{"status": "not_ready"}
```

---

## Authentication

### GET /api/v1/auth/login

Initiates OIDC login flow. Redirects the user to the identity provider (Zitadel).

- **Auth**: None
- **Rate limited**: Yes (public)

```bash
curl -v http://localhost:8080/api/v1/auth/login
# Returns 302 redirect to Zitadel authorize URL
```

**Notes**: Sets PKCE state/nonce cookies before redirecting.

### GET /api/v1/auth/callback

OIDC callback endpoint. Exchanges authorization code for tokens, creates/finds the identity record, and sets the session cookie.

- **Auth**: None (called by identity provider)
- **Rate limited**: Yes (public)
- **Query params**: `code`, `state` (set by the identity provider)

**Success**: Redirects to `/` (existing user) or `/onboarding` (new user without memberships).

**Error**: Returns 401 or 500 as JSON.

### POST /api/v1/auth/logout

Clears the session cookie.

- **Auth**: None (clears cookie regardless)

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout
```

**Response** `200`: Session cookie cleared.

### GET /api/v1/auth/me

Returns the current session user's profile.

- **Auth**: Session cookie (returns 401 if invalid/missing)

```bash
curl http://localhost:8080/api/v1/auth/me -b "mm_session=..."
```

**Response** `200`:
```json
{
  "subject": "user-zitadel-id",
  "identity_id": "uuid",
  "email": "user@example.com",
  "email_verified": true,
  "name": "Jane Doe",
  "given_name": "Jane",
  "family_name": "Doe",
  "roles": ["owner"],
  "has_memberships": true
}
```

**Response** `401`:
```json
{"error": "not authenticated"}
```

---

## Memberships

### GET /api/v1/memberships

List all organization memberships for the authenticated identity.

- **Auth**: Session cookie (SessionIdentityMiddleware only — no tenant context needed)

```bash
curl http://localhost:8080/api/v1/memberships -b "mm_session=..."
```

**Response** `200`:
```json
{"data": {"memberships": [{"membership_id": "uuid", "organization_id": "uuid"}]}}
```

---

## Onboarding

### POST /api/v1/onboarding

Complete first-time onboarding: creates organization, user membership, assigns owner role.

- **Auth**: Session cookie (read directly from cookie, not via protected middleware)
- **Rate limited**: Yes (public)

**Request**:
```json
{"organization_name": "Acme Corp"}
```

```bash
curl -X POST http://localhost:8080/api/v1/onboarding \
  -H "Content-Type: application/json" \
  -b "mm_session=..." \
  -d '{"organization_name": "Acme Corp"}'
```

**Response** `201`:
```json
{"data": {"organization_id": "uuid", "organization_name": "Acme Corp"}}
```

**Response** `409`:
```json
{"error": "identity already onboarded"}
```

---

## Organizations

### POST /api/v1/organizations

Create a new organization (standalone, without onboarding flow).

- **Auth**: None required (public)
- **Rate limited**: Yes (public)

**Request**:
```json
{"zitadel_org_id": "org-external-id", "name": "Acme Corp"}
```

**Response** `201`:
```json
{"data": {"id": "uuid", "organization_id": "org-external-id", "name": "Acme Corp"}}
```

**Response** `409`:
```json
{"error": "organization already exists"}
```

### GET /api/v1/organizations/dashboard

Full dashboard data for the current organization.

- **Auth**: Protected (session + membership + tenant)

**Response** `200`:
```json
{
  "data": {
    "organization": {"id": "uuid", "name": "Acme Corp"},
    "subscription": {"plan": "pro", "status": "active"},
    "usage": {"users": 5, "documents": 120, "api_requests": 5000, "storage": 1048576},
    "entitlements": [{"metric": "users", "used": 5, "limit": 50, "remaining": 45, "allowed": true}]
  }
}
```

### GET /api/v1/organizations/summary

Organization name and subscription summary.

- **Auth**: Protected

**Response** `200`:
```json
{"data": {"organization_id": "uuid", "organization_name": "Acme Corp", "plan": "pro", "status": "active"}}
```

### GET /api/v1/organizations/usage-summary

Current usage metrics.

- **Auth**: Protected

**Response** `200`:
```json
{"data": {"users": 5, "documents": 120, "api_requests": 5000, "storage": 1048576}}
```

---

## Users

### POST /api/v1/users

Register a new user (membership) within the current organization.

- **Auth**: Protected (session + membership + tenant)

**Request**:
```json
{"zitadel_user_id": "external-user-id", "email": "new@example.com"}
```

**Response** `201`:
```json
{"data": {"id": "uuid", "email": "new@example.com"}}
```

**Response** `409`:
```json
{"error": "user already exists"}
```

---

## RBAC

### GET /api/v1/roles

List all roles for the current organization.

- **Auth**: Protected

**Response** `200`:
```json
{"data": [{"id": "uuid", "name": "owner", "permissions": [{"id": "uuid", "name": "billing.write"}]}]}
```

### POST /api/v1/roles

Create a custom role.

- **Auth**: Protected
- **Permission**: `settings.write`
- **Rate limited**: Yes (authenticated)

**Request**:
```json
{"name": "editor", "permissions": ["users.read", "settings.read"]}
```

**Response** `201`:
```json
{"data": {"id": "uuid", "name": "editor", "permissions": [{"id": "uuid", "name": "users.read"}]}}
```

### GET /api/v1/permissions

List all available permissions.

- **Auth**: Protected

**Response** `200`:
```json
{"data": [{"id": "uuid", "name": "users.read"}, {"id": "uuid", "name": "billing.write"}]}
```

### POST /api/v1/users/{id}/roles

Assign a role to a user.

- **Auth**: Protected
- **Permission**: `settings.write`
- **Rate limited**: Yes (authenticated)
- **Path params**: `id` — user UUID

**Request**:
```json
{"role_id": "uuid"}
```

**Response** `201`:
```json
{"data": {"id": "uuid", "user_id": "uuid", "role_id": "uuid"}}
```

---

## Billing

### GET /api/v1/billing

Get the current organization's subscription.

- **Auth**: Protected
- **Permission**: `billing.read`

**Response** `200`:
```json
{
  "data": {
    "id": "uuid", "organization_id": "uuid", "provider": "dodo",
    "plan": "pro", "status": "active", "current_period_end": "2025-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-06-01T00:00:00Z"
  }
}
```

### POST /api/v1/billing

Create a subscription record.

- **Auth**: Protected
- **Permission**: `billing.write`

**Request**:
```json
{"provider": "dodo", "plan": "pro", "status": "active"}
```

**Response** `201`: Same shape as GET /api/v1/billing.

### PATCH /api/v1/billing/{id}

Update a subscription.

- **Auth**: Protected
- **Permission**: `billing.write`
- **Path params**: `id` — subscription UUID

**Request**:
```json
{"plan": "enterprise", "status": "active", "current_period_end": "2025-07-01T00:00:00Z"}
```

### POST /api/v1/billing/checkout

Create a checkout session with the payment provider.

- **Auth**: Protected
- **Permission**: `billing.write`
- **Rate limited**: Yes (authenticated)

**Request**:
```json
{"plan": "pro"}
```

**Response** `200`:
```json
{"data": {"checkout_url": "https://checkout.dodopayments.com/..."}}
```

### GET /api/v1/billing/subscription

Get subscription details (plan, status, provider, period end).

- **Auth**: Protected
- **Permission**: `billing.read`

**Response** `200`:
```json
{"data": {"plan": "pro", "status": "active", "provider": "dodo", "current_period_end": "2025-01-01T00:00:00Z"}}
```

### GET /api/v1/billing/usage

Get usage metrics for the organization.

- **Auth**: Protected
- **Permission**: `billing.read`

**Response** `200`:
```json
{"data": {"users": 5, "documents": 120, "api_requests": 5000, "storage": 1048576}}
```

### GET /api/v1/billing/entitlements

Get entitlement limits and current usage.

- **Auth**: Protected
- **Permission**: `billing.read`

**Response** `200`:
```json
{
  "data": {
    "entitlements": [
      {"metric": "users", "used": 5, "limit": 50, "remaining": 45, "allowed": true},
      {"metric": "documents", "used": 120, "limit": 1000, "remaining": 880, "allowed": true}
    ]
  }
}
```

### POST /api/v1/billing/webhook

Receive payment provider webhook events (Dodo Payments).

- **Auth**: Webhook signature verification (headers: `webhook-id`, `webhook-signature`, `webhook-timestamp`)
- **Rate limited**: Yes (webhook)

**Notes**: This endpoint is called by the payment provider, not by the frontend. The webhook body must contain `metadata.organization_id` to associate the event with a tenant.

**Response** `200`:
```json
{"received": true}
```

---

## Invitations

### POST /api/v1/invitations

Create an invitation to join the organization.

- **Auth**: Protected (session + membership + tenant)

**Request**:
```json
{"email": "invited@example.com", "role": "member"}
```

**Response** `201`:
```json
{"data": {"token": "invite-token-string", "invite_url": "/invite/invite-token-string"}}
```

### POST /api/v1/invitations/accept

Accept an invitation using a token.

- **Auth**: Session cookie (read directly from cookie)
- **Rate limited**: Yes (public)

**Request**:
```json
{"token": "invite-token-string"}
```

**Response** `200`:
```json
{"data": {"organization_id": "uuid"}}
```

**Error responses**:
- `404`: invitation not found
- `400`: invitation expired
- `403`: email does not match invitation
- `409`: invitation already accepted

---

## Audit

### GET /api/v1/audit

List audit logs for the current organization.

- **Auth**: Protected
- **Permission**: `audit.read`
- **Query params**: `limit` (default 50), `offset` (default 0)

```bash
curl "http://localhost:8080/api/v1/audit?limit=20&offset=0" -b "mm_session=..."
```

**Response** `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "action": "role_assigned",
      "entity_type": "user_role",
      "entity_id": "uuid",
      "user_id": "uuid",
      "created_at": "2024-06-01T12:00:00Z",
      "metadata": {"organization_id": "uuid"}
    }
  ]
}
```

---

## Development-Only Endpoints

### GET /api/v1/token

Generate a development JWT token (only available when `APP_ENV=development`).

- **Auth**: None

```bash
curl http://localhost:8080/api/v1/token
```

### GET /api/v1/ping

Simple connectivity test.

```bash
curl http://localhost:8080/api/v1/ping
# Response: pong (plain text)
```

---

## Metrics

### GET /metrics

Prometheus metrics endpoint.

- **Auth**: Bearer token (`METRICS_TOKEN` environment variable)
- **Not** under `/api/v1`

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/metrics
```
