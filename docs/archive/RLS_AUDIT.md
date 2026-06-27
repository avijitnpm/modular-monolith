
  I now have all the information needed for the full report.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  RLS Inventory

  Task 1: Every Repository Method Touching RLS-Protected Tables

  ┌──────────────────────────────┬──────────────────────────────┬─────────────────────────────────────┬────────────────────────┐
  │ Repository                   │ Method                       │ Table(s)                            │ Query Type             │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ billing.Repository           │ GetSubscription              │ subscriptions                       │ SELECT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ billing.Repository           │ CreateSubscription           │ subscriptions                       │ INSERT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ billing.Repository           │ UpdateSubscription           │ subscriptions                       │ UPDATE                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ billing.Repository           │ UpsertSubscriptionByProvider │ subscriptions                       │ INSERT/UPDATE (upsert) │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ ListRoles                    │ roles, role_permissions             │ SELECT (JOIN)          │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ CreateRole                   │ roles, role_permissions             │ INSERT (tx)            │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ BootstrapDefaultRoles        │ roles, role_permissions             │ INSERT (tx)            │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ AssignRoleToUser             │ user_roles                          │ INSERT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ RemoveRoleFromUser           │ user_roles                          │ DELETE                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.Repository              │ UserHasPermission            │ users, user_roles, role_permissions │ SELECT (JOIN)          │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ repository.Repository        │ FindUserByZitadelUserID      │ users                               │ SELECT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ repository.Repository        │ CreateUser                   │ users                               │ INSERT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ repository.Repository        │ CreateOrganization           │ organizations                       │ INSERT                 │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ repository.Repository        │ CreateOrganizationTx         │ organizations                       │ INSERT (external tx)   │
  ├──────────────────────────────┼──────────────────────────────┼─────────────────────────────────────┼────────────────────────┤
  │ rbac.BootstrapDefaultRolesTx │ (function)                   │ roles, role_permissions             │ INSERT (external tx)   │
  └──────────────────────────────┴──────────────────────────────┴─────────────────────────────────────┴────────────────────────┘

  NOT RLS-protected (safe regardless):

  | rbac.Repository | ListPermissions | permissions | SELECT |

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Production Call Graph

  Task 2: Every Production Call Path

  Protected routes (auth + tenant context set via middleware):

  GET /billing
  → billing.Handler.GetBilling
  → billing.Service.GetSubscription
  → billing.Repository.GetSubscription [subscriptions]

  POST /billing
  → billing.Handler.CreateBilling
  → billing.Service.CreateSubscription
  → billing.Repository.CreateSubscription [subscriptions]

  PATCH /billing/{id}
  → billing.Handler.UpdateBilling
  → billing.Service.UpdateSubscription
  → billing.Repository.UpdateSubscription [subscriptions]

  POST /billing/checkout
  → billing.Handler.CreateCheckout
  → billing.Service.CreateCheckoutSession
  → (no DB query on subscriptions)

  GET /roles
  → rbac.Handler.ListRoles
  → rbac.Service.ListRoles
  → rbac.Repository.ListRoles [roles, role_permissions]

  POST /roles
  → rbac.Handler.CreateRole
  → rbac.Service.CreateRole
  → rbac.Repository.CreateRole [roles, role_permissions] (tx)

  POST /users/{id}/roles
  → rbac.Handler.AssignRoleToUser
  → rbac.Service.AssignRoleToUser
  → rbac.Repository.AssignRoleToUser [user_roles]

  POST /users
  → users.Handler.RegisterUser
  → service.Service.RegisterUser
  → repository.Repository.CreateUser [users] (tx)

  GET /permissions (RBAC middleware check on ALL protected routes)
  → rbac.RequirePermission middleware
  → rbac.Service.UserHasPermission
  → rbac.Repository.UserHasPermission [users, user_roles, role_permissions]

  Public routes (no auth, no tenant context):

  POST /organizations
  → organizations.Handler.CreateOrganization
  → service.Service.RegisterOrganization
  → repository.Repository.CreateOrganizationTx [organizations] (tx)
  → rbac.BootstrapDefaultRolesTx [roles, role_permissions] (same tx)

  POST /admin/bootstrap-rbac
  → rbac.Handler.BootstrapRBAC
  → rbac.Service.BootstrapDefaultRoles
  → rbac.Repository.BootstrapDefaultRoles [roles, role_permissions] (tx)

  POST /billing/webhook
  → billing.Handler.HandleWebhook
  → billing.Service.ProcessWebhookEvent
  → billing.Repository.GetSubscription [subscriptions]
  → billing.Repository.UpsertSubscriptionByProvider [subscriptions]

  GET /auth/callback
  → authflow.Handler.Callback
  → service.Service.ProvisionAuthenticatedUser
  → repository.Repository.FindUserByZitadelUserID [users]
  → repository.Repository.CreateUser [users]

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Transaction Map

  Task 3: Transaction Boundaries

  ┌─────────────────────────────────────────────────┬───────────────────────────────────────────────────────┬───────────────────────────────────────────────────────────────────────────────┐
  │ Repository                                      │ Method                                                │ Execution Context                                                             │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository.GetSubscription              │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository.CreateSubscription           │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository.UpdateSubscription           │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository.UpsertSubscriptionByProvider │ Direct pool (r.DB.Exec)                               │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.ListRoles                       │ Direct pool (r.DB.Query)                              │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.CreateRole                      │ Internal tx (r.withTx)                                │ Self-managed transaction                                                      │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.BootstrapDefaultRoles           │ Internal tx (r.withTx)                                │ Self-managed transaction                                                      │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.BootstrapDefaultRolesTx                    │ External tx passed in                                 │ Caller-managed transaction                                                    │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.AssignRoleToUser                │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.RemoveRoleFromUser              │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository.UserHasPermission               │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository.FindUserByZitadelUserID   │ Direct pool (r.DB.QueryRow)                           │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository.CreateUser                │ Direct pool (r.DB.QueryRow)                           │ Called inside Service.WithTransaction but query uses r.DB not the tx — tx is  │
  │                                                 │                                                       │ unused                                                                        │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository.CreateOrganization        │ Direct pool (delegates to CreateOrganizationTx(r.DB)) │ No transaction                                                                │
  ├─────────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository.CreateOrganizationTx      │ External tx passed in                                 │ Caller-managed transaction                                                    │
  └─────────────────────────────────────────────────┴───────────────────────────────────────────────────────┴───────────────────────────────────────────────────────────────────────────────┘

  Critical observation on RegisterUser: The service wraps in WithTransaction, but CreateUser uses r.DB.QueryRow directly — not the transaction object. The tx is passed but ignored.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Architectural Exceptions

  Task 4: RLS Incompatibilities

  1. Organization Bootstrap (CHICKEN-AND-EGG)

  Path: POST /organizations → RegisterOrganization

  - CreateOrganizationTx INSERTs into organizations (RLS-protected)
  - BootstrapDefaultRolesTx INSERTs into roles and role_permissions (RLS-protected)
  - Both run inside one transaction
  - The tenant being created does not yet exist when the INSERT fires
  - RLS WITH CHECK (organization_id = current_setting(...)) requires the setting to match the org being inserted
  - SET LOCAL must be set BEFORE these inserts within the same transaction
  - This is the only path that creates the first record for a tenant

  2. Auth Lookup (NO TENANT KNOWN)

  Path: GET /auth/callback → ProvisionAuthenticatedUser

  - FindUserByZitadelUserID queries users WHERE zitadel_user_id = $1 (no org_id filter)
  - RLS on users requires app.current_organization_id to be set
  - At this point in the flow, the organization_id is not yet confirmed — it comes from the token claims but hasn't been validated against a local record
  - This is a cross-tenant lookup by design — the system needs to find which org a user belongs to
  - CreateUser (fallback) also needs to INSERT into users when the user doesn't exist

  3. Billing Webhook (NO AUTH CONTEXT)

  Path: POST /billing/webhook → HandleWebhook

  - No auth middleware (public route, signature-verified)
  - No middleware.TenantContext in the chain
  - Organization ID comes from webhook payload metadata, not from Go context
  - GetSubscription and UpsertSubscriptionByProvider query subscriptions (RLS-protected)
  - No mechanism to set app.current_organization_id exists in this path
  - Must either bypass RLS or set tenant context from the payload-provided org_id

  4. RBAC Bootstrap (NO AUTH CONTEXT)

  Path: POST /admin/bootstrap-rbac → BootstrapRBAC

  - Public route, no auth
  - Organization ID comes from request body
  - Inserts into roles and role_permissions (RLS-protected)
  - Same chicken-and-egg as org creation — but here the org already exists, so SET LOCAL is viable

  5. RBAC Permission Check (MIDDLEWARE — EVERY PROTECTED ROUTE)

  Path: Every rbac.RequirePermission middleware invocation

  - UserHasPermission JOINs across users, user_roles, role_permissions
  - All three tables have RLS
  - This runs AFTER middleware.TenantContext sets org_id in Go context
  - But the Go context org_id is never propagated to PostgreSQL app.current_organization_id
  - If RLS activates, this query returns zero rows → every authenticated user gets 403 on every permission-gated route

  6. Background Jobs

  None found.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Method Classification

  Task 5: Classification

  ┌──────────────────────────────┬──────────────────────────────┬──────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Repository                   │ Method                       │ Classification       │ Reason                                                                                     │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository           │ GetSubscription              │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository           │ CreateSubscription           │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository           │ UpdateSubscription           │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ billing.Repository           │ UpsertSubscriptionByProvider │ SPECIAL CASE         │ Called from webhook (no auth context) — org_id from payload                                │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ ListRoles                    │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ CreateRole                   │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ BootstrapDefaultRoles        │ BYPASS REQUIRED      │ Public route, no auth; inserts for an org not yet in context                               │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.BootstrapDefaultRolesTx │ (function)                   │ BYPASS REQUIRED      │ Called during org creation — must SET LOCAL inside same tx                                 │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ AssignRoleToUser             │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ RemoveRoleFromUser           │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ UserHasPermission            │ NEEDS TENANT CONTEXT │ Protected route, org_id available                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ rbac.Repository              │ ListPermissions              │ SAFE                 │ permissions table has no RLS                                                               │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository        │ FindUserByZitadelUserID      │ BYPASS REQUIRED      │ Auth callback — org unknown at lookup time                                                 │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository        │ CreateUser                   │ SPECIAL CASE         │ Two paths: (1) protected route → needs tenant context; (2) auth callback → bypass required │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository        │ CreateOrganization           │ BYPASS REQUIRED      │ Public route, creating new tenant                                                          │
  ├──────────────────────────────┼──────────────────────────────┼──────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────┤
  │ repository.Repository        │ CreateOrganizationTx         │ BYPASS REQUIRED      │ Same — runs inside tx creating the new org                                                 │
  └──────────────────────────────┴──────────────────────────────┴──────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────┘

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Implementation Roadmap

  Task 6: Phased Plan

  Phase 9B — Infrastructure (Tenant-Aware Query Execution)

  Objective: Create the mechanism to set SET LOCAL app.current_organization_id within the same transaction/connection as the actual query.

  Files to modify:

  - internal/database/tenant.go — redesign to work within a shared tx (not its own committed tx)
  - internal/repository/tenant.go — rewrite WithTenantContext to return a tenant-scoped connection/tx usable by queries
  - internal/repository/repository.go — add a TenantQuery or WithOrg method that wraps queries in a tenant-scoped tx

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Phase 9C — Protected Route Activation

  Objective: Activate RLS for all authenticated routes where org_id is available from middleware.

  Files to modify:

  - internal/modules/billing/repository.go — GetSubscription, CreateSubscription, UpdateSubscription must execute within tenant-scoped connection
  - internal/modules/rbac/repository.go — ListRoles, AssignRoleToUser, RemoveRoleFromUser, UserHasPermission, CreateRole must use tenant-scoped connection
  - internal/repository/user_repository.go — CreateUser (protected-route path only)
  - internal/middleware/tenant.go — potentially propagate org_id to a mechanism accessible by repositories

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Phase 9D — Bypass Paths

  Objective: Handle architectural exceptions that cannot use RLS (org creation, auth lookup, webhooks).

  Files to modify:

  - internal/repository/organization_repository.go — CreateOrganizationTx must SET LOCAL the new org_id inside the existing tx before INSERT
  - internal/service/organization_service.go — pass org_id to tx for SET LOCAL before CreateOrganizationTx
  - internal/modules/rbac/repository.go — BootstrapDefaultRolesTx must have SET LOCAL called before it in the same tx; BootstrapDefaultRoles (public endpoint) needs bypass or SET LOCAL from
  body param
  - internal/repository/user_repository.go — FindUserByZitadelUserID needs bypass (superuser connection or SECURITY DEFINER function or query via a non-RLS-restricted role)
  - internal/modules/billing/webhook_handler.go — must establish tenant context from payload org_id before repository calls
  - internal/modules/billing/repository.go — UpsertSubscriptionByProvider must support webhook path (SET LOCAL from handler-provided org_id)

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Phase 9E — Database Role Separation & Verification

  Objective: Ensure the connection role is NOT the table owner (otherwise RLS is bypassed by default). Add FORCE ROW LEVEL SECURITY or use a non-owner role for application queries.

  Files to modify:

  - migrations/ — new migration to either: (a) ALTER TABLE ... FORCE ROW LEVEL SECURITY on all 6 tables, or (b) create an application-level role with restricted privileges
  - internal/database/postgres.go — potentially configure separate connection pools (privileged for bypass paths, restricted for normal queries)
  - docker-compose.yml / .env — database role configuration

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Risk Assessment

  ┌───────────────────────────────────────────────┬──────────┬─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │ Risk                                          │ Severity │ Description                                                                                                                 │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Total application breakage if RLS activates   │ CRITICAL │ Every query against RLS tables returns 0 rows or fails INSERT WITH CHECK. All protected routes return 500/empty. RBAC       │
  │ before query refactor                         │          │ middleware denies all requests.                                                                                             │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Auth callback breaks                          │ HIGH     │ FindUserByZitadelUserID returns no rows → all login flows fail → application inaccessible                                   │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Organization creation breaks                  │ HIGH     │ INSERT into organizations fails WITH CHECK → no new tenants can be created                                                  │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Webhook processing breaks                     │ MEDIUM   │ UpsertSubscriptionByProvider fails → subscription state becomes stale → billing desyncs                                     │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ RegisterUser tx doesn't use tx object         │ MEDIUM   │ CreateUser queries r.DB directly despite being wrapped in WithTransaction — this means the SET LOCAL in a tx won't scope to │
  │                                               │          │ the actual query. Must refactor to use the tx object.                                                                       │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Connection role may be table owner            │ HIGH     │ If the app connects as the table owner (common in dev), RLS is silently bypassed regardless of implementation. Need FORCE   │
  │                                               │          │ ROW LEVEL SECURITY or role separation.                                                                                      │
  ├───────────────────────────────────────────────┼──────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ Test infrastructure                           │ LOW      │ Integration tests use context.Background() and direct pool — need test helpers for tenant context                           │
  └───────────────────────────────────────────────┴──────────┴─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  Recommended order: 9E first (verify RLS is enforceable at the DB level) → 9B (infrastructure) → 9C (standard paths) → 9D (exceptions). Activating 9C before 9D would break
  auth/org-creation/webhooks.


