# Phase 14D-D: Tenant Provisioning Architecture Recommendation

**Date:** 2026-06-24
**Status:** Proposal
**Scope:** OAuth callback → user provisioning → organization membership

---

## 1. Current State (Broken)

```
Login → Zitadel OIDC → Callback → ValidateToken → normalizeUser
  → ProvisionAuthenticatedUser(subject, email, organizationID)
    → FindUserByZitadelUserID(orgID, userID)  ← FAILS if orgID == ""
    → CreateUser(orgID, userID, email)         ← FAILS if orgID == ""
```

**Root cause:** `database.WithTenantQuery()` returns an error if `organizationID == ""`. Every user query is gated by RLS. The OAuth callback passes `organizationID` extracted from OIDC claims, but Zitadel does not guarantee this claim.

**Fundamental flaw:** The system treats the IdP as the source of truth for organization membership. In a multi-tenant SaaS, organization membership is application-owned data.

---

## 2. Design Principles

1. **Identity ≠ tenancy.** Authentication proves who you are. Organization membership is a separate concern.
2. **The application database is the source of truth** for organization membership.
3. **RLS must never be bypassed** for tenant-scoped data.
4. **A user without an organization is a valid state** — they are authenticated but not yet tenanted.
5. **The onboarding funnel must be explicit** — not hidden inside the OAuth callback.

---

## 3. Proposed Data Model Change

### 3.1 New table: `identities`

```sql
CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,          -- 'zitadel'
    provider_user_id TEXT NOT NULL,   -- zitadel subject
    email TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    display_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_user_id)
);
```

**No RLS on `identities`.** This table is tenant-agnostic. It stores "this person exists" — nothing more.

### 3.2 Existing `users` table remains tenant-scoped

The `users` table retains `organization_id NOT NULL` and its RLS policy. A row in `users` means: "this identity is a member of this organization."

### 3.3 Link table: `identity_organization_memberships`

```sql
CREATE TABLE identity_organization_memberships (
    identity_id UUID NOT NULL REFERENCES identities(id),
    organization_id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'active',  -- active, suspended, invited
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (identity_id, organization_id)
);
```

**No RLS on this table.** It is queried by the auth layer to resolve "which organizations does this identity belong to?" before a tenant context is set.

---

## 4. Proposed Flow

### 4A. First-Time Login (user not found)

```
Callback → ValidateToken → normalizeUser
  → FindIdentityByProvider(provider, subject)
    → NOT FOUND
  → CreateIdentity(provider, subject, email)
  → Create session with: identity_id, email, organizations=[]
  → Redirect to /onboarding
```

**No organization context is set.** The session is valid but un-tenanted. The frontend detects `organizations: []` and shows the onboarding screen.

### 4B. Existing User Login (already provisioned, has org)

```
Callback → ValidateToken → normalizeUser
  → FindIdentityByProvider(provider, subject)
    → FOUND (identity_id = X)
  → ListMemberships(identity_id)
    → [{org_id: "org-abc", user_id: "user-123", status: "active"}]
  → Create session with: identity_id, active_organization="org-abc", user_id="user-123"
  → Redirect to /dashboard
```

### 4C. User Without Organization

Same as 4A post-login. Session exists, no `active_organization`. All tenant-scoped API routes reject the request with `403` or redirect to `/onboarding`.

The onboarding route allows:
- **Create organization** → calls `RegisterOrganization()`, creates membership, assigns `owner` role
- **Join via invitation** → accepts a pending invite, creates membership

### 4D. User Invited to Organization

```
Admin calls POST /api/v1/organizations/:org_id/invitations
  → Creates invitation record (email, org_id, role, expires_at)
  → (Optional: sends email)

Invited user logs in:
  → FindIdentityByProvider → found or created
  → ListPendingInvitations(email)
    → [{org_id: "org-abc", role: "member"}]
  → Frontend shows: "You've been invited to Acme Corp"
  → User accepts → CreateUser(org_id, ...) + CreateMembership + AssignRole
```

---

## 5. Architectural Decisions

### 5.1 Should `organization_id` be nullable during provisioning?

**No.** The `users` table must retain `organization_id NOT NULL`. Making it nullable would:
- Break RLS policies on all tenant-scoped tables
- Require NULL-handling in every query
- Create orphan rows with no isolation context

Instead, introduce the `identities` table as the pre-tenant identity store.

### 5.2 Should an onboarding route exist?

**Yes.** Route: `GET /onboarding` (frontend) + `POST /api/v1/onboarding/organization` (backend).

The onboarding route is accessible to authenticated users with no organization. It must:
- Allow creating a new organization (which triggers `RegisterOrganization` + user creation + `owner` role assignment)
- Allow accepting a pending invitation
- Be the only tenant-scoped action available to un-tenanted users

### 5.3 Should OAuth callback create identity only?

**Yes.** The callback's responsibility becomes:
1. Validate the OIDC token
2. Upsert the identity record (tenant-agnostic)
3. Resolve organization memberships
4. Create the session
5. Redirect based on membership state

It must **not** create users or organizations. Those are explicit actions.

### 5.4 Should organization assignment happen afterward?

**Yes.** Organization assignment happens:
- During onboarding (user creates or joins an org)
- Via invitation acceptance
- Via admin API (for programmatic provisioning)

Never during the OAuth callback.

---

## 6. Session Structure (Revised)

```go
type Session struct {
    IdentityID         string   `json:"identity_id"`
    Email              string   `json:"email"`
    DisplayName        string   `json:"display_name,omitempty"`
    ActiveOrganization string   `json:"active_organization,omitempty"` // "" = no org
    UserID             string   `json:"user_id,omitempty"`             // "" = no org membership
    Organizations      []string `json:"organizations"`                 // all memberships
    ExpiresAt          int64    `json:"expires_at"`
}
```

**Routing logic:**
- `ActiveOrganization == ""` → only `/onboarding`, `/api/v1/auth/*`, `/api/v1/onboarding/*` accessible
- `ActiveOrganization != ""` → full tenant-scoped API access via existing RLS path

---

## 7. Impact on Existing Systems

| System | Impact | Change Required |
|--------|--------|-----------------|
| **RLS** | None. `users` table keeps `organization_id NOT NULL`. RLS policies unchanged. | No |
| **RBAC** | `user_roles` still references `users.id`. Role assignment happens after user creation. | No schema change. Logic change: assign role during onboarding, not callback. |
| **Billing** | `subscriptions` keyed on `organization_id`. Unaffected — org must exist before subscription. | No |
| **Usage tracking** | `usage_counters` keyed on `organization_id`. Unaffected. | No |
| **Entitlements** | Entitlements are org-scoped. Un-tenanted users have no entitlements. | No schema change. Add guard: reject if no active org. |
| **Audit logging** | Audit events require `organization_id`. Identity-level events (login, logout) use a system/identity audit scope. | Minor: add identity-scoped audit events. |
| **Middleware** | `TenantContext` middleware requires `AuthenticatedUser.OrganizationID`. Must allow pass-through for onboarding routes. | Route-level: onboarding routes skip tenant middleware. |

---

## 8. Migration Strategy

### Phase 1: Add `identities` table + `identity_organization_memberships`
- New migration. No existing table changes.
- Backfill: for every existing `users` row, create a corresponding `identities` row using `zitadel_user_id`.

### Phase 2: Modify OAuth callback
- Replace `ProvisionAuthenticatedUser` with `ProvisionIdentity` (tenant-agnostic).
- Resolve memberships from `identity_organization_memberships`.
- Redirect to `/onboarding` if no memberships.

### Phase 3: Add onboarding endpoints
- `POST /api/v1/onboarding/organization` — create org + self-assign owner
- `POST /api/v1/onboarding/accept-invitation` — accept pending invite

### Phase 4: Add invitation system
- `POST /api/v1/organizations/:org_id/invitations` — create invite
- `DELETE /api/v1/organizations/:org_id/invitations/:id` — revoke
- Invitation table with email, org_id, role, token, expires_at

---

## 9. Security Considerations

1. **Identity table has no RLS** — it stores only IdP-sourced data (subject, email). No business data leaks.
2. **Membership resolution happens before tenant context is set** — uses direct query (no RLS), returns only org IDs the identity belongs to.
3. **Onboarding routes must be rate-limited** — prevent org-creation spam.
4. **Invitation tokens must be single-use, time-limited, and cryptographically random.**
5. **Organization switching** — when a user has multiple memberships, the session tracks `active_organization`. Switching orgs re-sets the tenant context. No cross-tenant data leakage.

---

## 10. What Does NOT Change

- `database.WithTenantQuery()` — still requires non-empty `organizationID`
- `users.organization_id` — remains `NOT NULL`
- RLS policies on all tenant tables — unchanged
- `RegisterOrganization()` — same transaction (create org + bootstrap RBAC)
- RBAC role/permission structure — unchanged
- Billing, usage, entitlements schemas — unchanged

---

## 11. Summary

| Question | Answer |
|----------|--------|
| Make `organization_id` nullable? | **No** |
| Add onboarding route? | **Yes** |
| OAuth callback creates identity only? | **Yes** |
| Organization assignment happens afterward? | **Yes** |
| New table for pre-tenant identity? | **Yes** (`identities`) |
| Existing RLS/RBAC/billing affected? | **No** |
