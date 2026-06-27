# Phase 17D — Transactional Hardening & Failure Recovery

**Date:** 2026-06-24
**Status:** Complete

---

## Transaction Boundaries

### Onboarding Flow

| Step | Operation | Transaction Boundary | Rollback Behavior |
|------|-----------|---------------------|-------------------|
| 1 | Check existing membership | Single query, no tx | N/A |
| 2 | RegisterOrganization | **Single tx**: org INSERT + role bootstrap (4 roles + permissions) | Full rollback on any failure |
| 3 | CreateUserWithRole | **Single tx**: user INSERT + role assignment INSERT | Full rollback — no orphaned user without role |
| 4 | Audit log | Best-effort, fire-and-forget | Does not affect success |

**Before 17D:** Steps 2, user creation, and role assignment were 3 separate transactions. A failure between any step left inconsistent state.

**After 17D:** Only 2 critical transactions remain:
- Tx1: org + roles (already atomic via `RegisterOrganization`)
- Tx2: user + role assignment (new `CreateMembershipWithRole`)

**Unavoidable boundary:** Tx1 and Tx2 cannot be combined because the org creation uses `WithTransaction` (which begins its own pool-level tx) while the user creation uses `WithTenantQuery` (which requires tenant context set). These are different transaction scopes.

**Recovery:** If Tx1 succeeds but Tx2 fails → orphaned empty org. On retry, a new org is created (fresh UUID), user gets the new one. Old org has no members and no effect on the system.

### Invitation Acceptance Flow

| Step | Operation | Transaction Boundary | Rollback Behavior |
|------|-----------|---------------------|-------------------|
| 1 | Get invitation by token | Single query | N/A |
| 2 | Validate (expiry, email, not accepted) | In-memory checks | N/A |
| 3 | CreateUserWithRole | **Single tx**: user INSERT + role assignment | Full rollback |
| 4 | Mark invitation accepted | Single UPDATE | N/A |
| 5 | Audit log | Best-effort | N/A |

**Unavoidable boundary:** Step 3 (user+role in target org tx) and Step 4 (invitation table update, different org context) cannot share a transaction due to RLS tenant context differences.

**Recovery:** If step 3 succeeds but step 4 fails → user has membership but invitation shows as unaccepted. Retry hits `ErrUserAlreadyExists` → mapped to 409 (idempotent).

---

## Idempotency Guarantees

| Flow | Retry Behavior | HTTP Response |
|------|---------------|---------------|
| Onboarding (after success) | `HasMembership` returns true | 409 "identity already onboarded" |
| Onboarding (partial: org created, user failed) | New org created, user retried | 201 (success on retry) |
| Invitation accept (after success) | `AcceptedAt != nil` check | 409 "invitation already accepted" |
| Invitation accept (user created, mark failed) | `ErrUserAlreadyExists` on user creation | 409 "invitation already accepted" |

---

## Defects Discovered & Fixed

### Defect 1: User creation and role assignment not atomic

**Severity:** High
**Impact:** User could exist without role → no permissions → unusable account

**Fix:** Added `Repository.CreateMembershipWithRole()` — single INSERT for user + INSERT for role assignment in one `WithTenantQuery` transaction.

**Files:** `internal/repository/user_repository.go`, `internal/service/user_service.go`

### Defect 2: Invitation retry returns 500 instead of 409

**Severity:** Medium
**Impact:** Retrying an already-accepted invitation where mark_accepted failed returns confusing 500 error

**Fix:** Added `ErrUserAlreadyExists` case to invitation handler error switch → returns 409.

**File:** `internal/modules/invitations/handler.go`

### Defect 3: Redundant role bootstrap in onboarding

**Severity:** Low (no functional impact)
**Impact:** `RegisterOrganization` already bootstraps roles, then `onboardingRoleAdapter.AssignOwnerRole` bootstraps again.

**Status:** Not fixed — `ON CONFLICT DO NOTHING` makes it idempotent. The redundancy adds ~5ms latency but no correctness issue. Left as-is because the atomic `CreateMembershipWithRole` path now bypasses the role adapter entirely.

---

## Fresh Deployment Validation

| Check | Mechanism | Status |
|-------|-----------|--------|
| Empty DB startup | `database.Migrate()` in `app/start.go` | ✅ |
| Migration tracking | `schema_migrations` table auto-created | ✅ |
| Migration ordering | Files sorted alphabetically (001_ through 016_) | ✅ |
| Tern separator handling | Only content before `---- create above / drop below ----` applied | ✅ |
| Permission seeding | Migration 008 inserts 7 permissions | ✅ |
| Re-run safety | Already-applied migrations skipped via `schema_migrations` lookup | ✅ |
| Migration failure | Error logged + startup aborted | ✅ |
| DB unavailable | Connection fails → startup error returned | ✅ |

---

## Operational Readiness

| Feature | Implementation | Status |
|---------|---------------|--------|
| Liveness probe | `GET /health/live` → 200 | ✅ |
| Readiness probe | `GET /health/ready` → DB ping | ✅ |
| Simple health | `GET /health` → "ok" | ✅ |
| Startup logging | `logger.Info("server starting", "port", port)` | ✅ |
| Migration logging | Each applied file logged | ✅ |
| Migration failure | Logged as error, startup aborted | ✅ |
| Graceful shutdown | `http.ErrServerClosed` handled, OTEL shutdown | ✅ |
| Panic recovery | `middleware.Recovery` catches panics | ✅ |

---

## Remaining Operational Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Org leak on partial onboarding failure | Low | Empty orgs have no members; harmless. Could add cleanup job later. |
| Tx1/Tx2 boundary in onboarding | Medium | Cannot be combined due to different tenant contexts. Documented as known limitation. |
| No distributed tracing of tx boundaries | Low | OTEL tracing covers HTTP spans; DB tx spans not individually traced. |
| No retry mechanism for failed audit logs | Low | Audit is best-effort by design. Missing audit entries don't affect user flow. |

---

## Files Modified

| File | Change |
|------|--------|
| `internal/repository/user_repository.go` | Added `CreateMembershipWithRole` (atomic user + role) |
| `internal/service/user_service.go` | Added `RegisterMembershipWithRole` |
| `internal/modules/onboarding/service.go` | Added `UserCreatorWithRole` interface, uses atomic path |
| `internal/modules/invitations/service.go` | Added `UserCreatorWithRole` interface, uses atomic path |
| `internal/modules/invitations/handler.go` | Added `ErrUserAlreadyExists` → 409 mapping |
| `internal/router/routes.go` | Wired `UsersWithRole` adapters for onboarding + invitations |

---

## Test Results

```
$ go vet ./...          → clean
$ go test ./...         → 20 packages pass
$ go build ./...        → clean
```
