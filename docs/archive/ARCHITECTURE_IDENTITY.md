# Identity & Membership Architecture

## Layers

### Identity Layer (Global)

- Table: `identities`
- No RLS, no tenant scoping
- One row per person
- Keyed by: `identities.id` (UUID)
- Provider link: `identities.zitadel_user_id`

### Membership Layer (Tenant-scoped)

- Table: `users` (functions as memberships)
- RLS by `organization_id`
- One row per (identity, organization) pair
- Keyed by: `users.id` (UUID)
- FK: `users.identity_id → identities.id`

### Session Layer

- Encrypted cookie (`mm_session`)
- Contains: `IdentityID`, `Subject`, `Email`, `Name`
- Does NOT determine organization — that is resolved by middleware

### Authorization Layer

- RBAC operates on `membership_id` (users.id)
- Permission checks: `UserHasPermission(orgID, membershipID, permission)`
- Roles are org-scoped via `user_roles` table

## Request Flow

```
OIDC Login
  ↓
Token Validation (providers/identity)
  ↓
FindOrCreateIdentity (identity module)
  ↓
Session created with IdentityID
  ↓
[API Request]
  ↓
Auth Middleware → AuthenticatedUser (legacy) + IdentityContext + MembershipContext
  ↓
Tenant Middleware → OrganizationID (from MembershipContext, fallback AuthenticatedUser)
  ↓
RBAC Middleware → permission check using MembershipID
  ↓
Handler → tenant-scoped data access via RLS
```

## Context Values (Request Lifecycle)

| Context | Purpose | Source |
|---------|---------|--------|
| `AuthenticatedUser` | Legacy compatibility | Auth middleware (token claims) |
| `IdentityContext` | Who the person is | Identity resolver or session |
| `MembershipContext` | Which org they're accessing | Membership resolver |
| `OrganizationID` | RLS tenant key | Tenant middleware |

## Resolution Priority

Organization ID resolution:
1. `MembershipContext.OrganizationID` (canonical)
2. `AuthenticatedUser.OrganizationID` (legacy fallback)
3. Error

User ID for RBAC:
1. `MembershipContext.MembershipID` (canonical)
2. `AuthenticatedUser.UserID` (legacy fallback)

## Provider Abstraction

Only these packages may reference `zitadel_user_id`:
- `internal/providers/identity`
- `internal/modules/identity`
- `internal/modules/identityresolver`
- `internal/modules/authflow`

All domain services use:
- `identity_id` (from identities table)
- `membership_id` (users.id)

## Compatibility

`AuthenticatedUser` and `SessionUser.OrganizationID` remain for backward
compatibility with existing sessions and API token paths. They will be removed
in a future phase once all consumers are migrated.
