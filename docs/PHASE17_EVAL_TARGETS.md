# Phase 17 — Evaluation Targets

## Authentication

**Risk: Low**

- OIDC flow stable (Zitadel)
- Session encryption (AES-GCM) working
- PKCE + nonce validation in place

Known assumptions:
- Single identity provider (Zitadel)
- Session cookie is the only session mechanism for browser flows

## Session

**Risk: Low**

- Session now stores `IdentityID`
- `OrganizationID` field legacy — no longer drives routing
- Old sessions without `IdentityID` will fallback gracefully

Known migration point:
- Existing sessions in the wild won't have `IdentityID` until re-login

## Identity

**Risk: Low**

- `identities` table is global, no RLS
- `FindOrCreateIdentity` is idempotent (upsert on email/name changes)
- Provider abstraction via `identityresolver.Resolver` interface

## Membership

**Risk: Medium**

- `users` table serves as memberships
- `identity_id` FK established (Phase 16B)
- Global UNIQUE on `zitadel_user_id` and `email` still exists — blocks multi-org
- `CreateMembership` resolves `zitadel_user_id` via subquery for compat

Known migration point:
- UNIQUE constraints must be relaxed before multi-org is possible
- Table rename (`users` → `memberships`) deferred

## RBAC

**Risk: Low**

- Permission checks use `membership_id` (users.id)
- Roles are org-scoped
- Bootstrap creates owner/admin/member/viewer per org

Known assumptions:
- `user_roles.user_id` references `users.id` — stable
- Permission checks assume single active membership per request

## Onboarding

**Risk: Medium**

- Uses `identity_id` for `HasMembership` check
- Creates membership via `CreateMembership(orgID, identityID, email)`
- Handler resolves identity from session subject via `identityresolver`

Known assumptions:
- `HasMembership` returns true if ANY membership exists (blocks second-org onboarding)
- Must be policy-gated in future for multi-org

## Invitations

**Risk: Medium**

- Acceptance uses `identity_id` (resolved from session)
- Creates membership + assigns role
- Email match enforced

Known assumptions:
- Same UNIQUE constraint issue — invitation acceptance may fail for existing members
- No duplicate-membership detection beyond DB constraint

## Billing

**Risk: Low**

- Fully org-scoped (subscriptions, usage)
- No identity coupling
- Webhook path is provider-specific but domain-isolated

## Usage

**Risk: Low**

- Org-scoped counters
- No identity coupling
- Entitlements derived from subscription + usage

## Dashboard

**Risk: Low**

- Reads org context from tenant middleware
- Aggregates subscription + usage + entitlements
- No identity coupling

## Frontend Wiring

**Risk: Medium**

- `/auth/me` returns `SessionUser` including legacy `OrganizationID`
- Frontend may depend on `organization_id` from `/auth/me`
- Memberships endpoint (`GET /api/v1/memberships`) exists but not consumed yet

Known migration point:
- Frontend should migrate from `me.organization_id` to `/memberships` endpoint
- Org switcher UI needed for multi-org

## Summary

| Module | Risk | Blocker for Multi-Org |
|--------|------|----------------------|
| Authentication | Low | No |
| Session | Low | No |
| Identity | Low | No |
| Membership | Medium | UNIQUE constraints |
| RBAC | Low | No |
| Onboarding | Medium | HasMembership policy |
| Invitations | Medium | UNIQUE constraints |
| Billing | Low | No |
| Usage | Low | No |
| Dashboard | Low | No |
| Frontend | Medium | org_id dependency |
