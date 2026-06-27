# Module Guide

How to create a new module in this repository. Follow this guide exactly to maintain architectural consistency.

---

## Prerequisites

Before starting, you must understand:
- [ARCHITECTURE.md](ARCHITECTURE.md) — overall system design
- [ENGINEERING_RULES.md](ENGINEERING_RULES.md) — rules you must not break

---

## Folder Structure

Create your module at `internal/modules/<modulename>/`:

```
internal/modules/<modulename>/
├── handler.go          HTTP handlers
├── service.go          Business logic + interfaces
├── repository.go       Database access
├── models.go           Domain types (structs)
├── dto.go              Request/response types
├── errors.go           Sentinel errors
└── handler_test.go     Tests
```

Not every file is required. A simple module (like `health`) may only have `handler.go`. A complex module (like `billing`) may have additional files (`webhook.go`, `api_handler.go`).

---

## Step-by-Step

### 1. Create the migration

```bash
make migration name=<your_table_name>
```

This creates `migrations/NNN_<your_table_name>.sql`. Write your schema:

```sql
CREATE TABLE your_things (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_your_things_org ON your_things(organization_id);

-- MANDATORY: Enable Row Level Security
ALTER TABLE your_things ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_your_things
ON your_things
USING (
    organization_id = current_setting('app.current_organization_id', true)
)
WITH CHECK (
    organization_id = current_setting('app.current_organization_id', true)
);

---- create above / drop below ----

DROP TABLE IF EXISTS your_things;
```

**Critical**: Every tenant-scoped table MUST have `organization_id TEXT NOT NULL`, RLS enabled, and a `tenant_isolation_*` policy.

### 2. Define models

```go
// internal/modules/yourmod/models.go
package yourmod

import "time"

type Thing struct {
    ID             string
    OrganizationID string
    Name           string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### 3. Define errors

```go
// internal/modules/yourmod/errors.go
package yourmod

import "errors"

var (
    ErrThingNotFound      = errors.New("thing not found")
    ErrThingAlreadyExists = errors.New("thing already exists")
    ErrInvalidThing       = errors.New("invalid thing")
)
```

### 4. Create the repository

```go
// internal/modules/yourmod/repository.go
package yourmod

import (
    "context"
    "errors"

    "github.com/avijitnpm/modular-monolith/internal/database"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
    DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
    return &Repository{DB: db}
}

func (r *Repository) GetThing(ctx context.Context, organizationID, id string) (*Thing, error) {
    var thing Thing

    err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
        return tx.QueryRow(ctx,
            `SELECT id, organization_id, name, created_at, updated_at
             FROM your_things
             WHERE id = $1 AND organization_id = $2`,
            id, organizationID,
        ).Scan(&thing.ID, &thing.OrganizationID, &thing.Name, &thing.CreatedAt, &thing.UpdatedAt)
    })

    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrThingNotFound
    }

    return &thing, err
}
```

**Rules**:
- Always use `database.WithTenantQuery` for tenant-scoped data
- Always pass `organizationID` explicitly
- Always include `organization_id` in WHERE clauses (belt + suspenders with RLS)
- Return sentinel errors, not raw database errors

### 5. Create the service

```go
// internal/modules/yourmod/service.go
package yourmod

import (
    "context"
    "strings"

    "github.com/avijitnpm/modular-monolith/internal/audit"
)

// Store defines the repository interface (for testability)
type Store interface {
    GetThing(ctx context.Context, organizationID, id string) (*Thing, error)
    CreateThing(ctx context.Context, organizationID, name string) (*Thing, error)
}

type AuditLogger interface {
    Log(ctx context.Context, event *audit.Event) error
}

type Service struct {
    Repository Store
    Audit      AuditLogger
}

func NewService(repository Store, audit AuditLogger) *Service {
    return &Service{Repository: repository, Audit: audit}
}

func (s *Service) CreateThing(ctx context.Context, organizationID, name string) (*Thing, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return nil, ErrInvalidThing
    }

    thing, err := s.Repository.CreateThing(ctx, organizationID, name)
    if err != nil {
        return nil, err
    }

    // Audit log
    _ = s.Audit.Log(ctx, &audit.Event{
        OrganizationID: organizationID,
        Action:         "thing.created",
        EntityType:     "thing",
        EntityID:       thing.ID,
    })

    return thing, nil
}
```

**Rules**:
- Define a `Store` interface (not a concrete repo type) — enables testing
- Validate inputs in the service, not the handler
- Always emit audit events for state-changing operations
- Return domain errors, not HTTP status codes

### 6. Create the handler

```go
// internal/modules/yourmod/handler.go
package yourmod

import (
    "encoding/json"
    "errors"
    "net/http"

    appcontext "github.com/avijitnpm/modular-monolith/internal/context"
    "github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
    Service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{Service: service}
}

func (h *Handler) CreateThing(w http.ResponseWriter, r *http.Request) {
    organizationID, ok := appcontext.GetOrganizationID(r.Context())
    if !ok {
        response.InternalServerError(w, "organization context missing")
        return
    }

    var req CreateThingRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, "invalid request body")
        return
    }

    thing, err := h.Service.CreateThing(r.Context(), organizationID, req.Name)

    if errors.Is(err, ErrInvalidThing) {
        response.BadRequest(w, "name is required")
        return
    }

    if err != nil {
        response.InternalServerError(w, "failed to create thing")
        return
    }

    response.Created(w, thingResponse(thing))
}
```

**Rules**:
- Handlers are thin: decode → call service → encode response
- Always extract `organizationID` from context (set by TenantContext middleware)
- Use `pkg/response` helpers for consistent JSON responses
- Map domain errors to HTTP status codes here (not in service)
- Never log sensitive data in error messages returned to clients

### 7. Register routes

In `internal/router/routes.go`, add your module to the `registerRoutes` function:

```go
// Inside registerRoutes()
yourRepository := yourmod.NewRepository(service.Repository.DB)
yourService := yourmod.NewService(yourRepository, auditService)
yourHandler := yourmod.NewHandler(yourService)

// Inside the protected group:
protected.Get("/things", yourHandler.ListThings)
protected.With(
    rbac.RequirePermission(rbacService, "things.write"),
    middleware.AuthenticatedRateLimit(),
).Post("/things", yourHandler.CreateThing)
```

### 8. Add permission (if needed)

If your module requires a new permission, add it to a migration:

```sql
INSERT INTO permissions (name) VALUES ('things.read'), ('things.write')
ON CONFLICT (name) DO NOTHING;
```

Update the role bootstrap in `internal/modules/rbac/repository.go` (`BootstrapDefaultRolesTx`) to assign the new permission to appropriate default roles.

---

## DTO Conventions

```go
// dto.go
type CreateThingRequest struct {
    Name string `json:"name"`
}

type ThingResponse struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    CreatedAt string `json:"created_at"`
}

func thingResponse(t *Thing) ThingResponse {
    return ThingResponse{
        ID:        t.ID,
        Name:      t.Name,
        CreatedAt: t.CreatedAt.Format(time.RFC3339),
    }
}
```

---

## Cross-Module Communication

Modules CANNOT import each other. If your module needs data from another module, define an adapter interface in `routes.go`:

```go
// In routes.go
type thingBillingAdapter struct {
    store billing.Store
}

func (a *thingBillingAdapter) GetPlan(ctx context.Context, orgID string) (string, error) {
    sub, err := a.store.GetSubscription(ctx, orgID)
    if err != nil { return "", err }
    if sub == nil { return "", nil }
    return sub.Plan, nil
}
```

Then inject this adapter into your service via an interface it defines.

---

## Testing

```go
// handler_test.go or service_test.go
func TestCreateThing_EmptyName(t *testing.T) {
    svc := NewService(&mockStore{}, &mockAudit{})
    _, err := svc.CreateThing(context.Background(), "org-1", "")
    if !errors.Is(err, ErrInvalidThing) {
        t.Fatalf("expected ErrInvalidThing, got %v", err)
    }
}
```

For integration tests that need a real database, see existing `*_integration_test.go` files for patterns (testcontainers or test database with cleanup).

---

## Common Mistakes

| Mistake | Consequence | Fix |
|---------|-------------|-----|
| Querying pool directly (not `WithTenantQuery`) | Bypasses RLS, returns all tenants' data | Always use `WithTenantQuery` for tenant-scoped tables |
| Importing another module's package | Creates coupling, breaks modularity | Use adapter interfaces in `routes.go` |
| Business logic in handler | Untestable, duplicated if called from webhook/job | Move to service layer |
| Returning raw DB errors to client | Leaks schema details | Map to sentinel errors, return generic messages |
| Forgetting `organization_id` in WHERE | RLS is defense-in-depth, but don't rely on it alone | Always filter explicitly |
| Missing RLS on new table | Data leak across tenants | Every tenant table needs RLS + policy |
| Missing audit log | Compliance gap | Every state change should log |
| Hardcoding organization_id | Works in dev, fails in production | Always get from `appcontext.GetOrganizationID(ctx)` |

---

## Checklist for New Module

- [ ] Migration created with RLS policy
- [ ] Models defined
- [ ] Repository uses `WithTenantQuery`
- [ ] Service defines `Store` interface
- [ ] Service validates inputs
- [ ] Service emits audit events
- [ ] Handler extracts org_id from context
- [ ] Handler maps domain errors to HTTP responses
- [ ] Routes registered in `routes.go`
- [ ] Permission added (if gated)
- [ ] Tests written
- [ ] No cross-module imports


---

## Request Lifecycle Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Request                            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Global Middleware                                           │
│  RequestID → CORS → OTEL → Logging → Recovery → Security   │
│  → BodyLimit → Metrics                                      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Auth Middleware (protected routes only)                     │
│  SessionIdentityMiddleware                                  │
│    → decrypt mm_session cookie                              │
│    → set IdentityContext (identity_id, email, name)         │
│  ResolveMembershipMiddleware                                │
│    → query users table by identity_id                       │
│    → set MembershipContext (membership_id, organization_id) │
│  TenantContext                                              │
│    → extract organization_id → set in request context       │
│  [RequirePermission] (per-route)                            │
│    → check RBAC: user → role → permission                   │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Handler                                                    │
│  • Extract organization_id from context                     │
│  • Decode JSON request body                                 │
│  • Call service method                                      │
│  • Map errors to HTTP status codes                          │
│  • Encode JSON response                                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Service                                                    │
│  • Validate business rules                                  │
│  • Orchestrate operations                                   │
│  • Call repository                                          │
│  • Emit audit event                                         │
│  • Return domain model or sentinel error                    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Repository                                                 │
│  database.WithTenantQuery(pool, ctx, orgID, func(tx) {      │
│    • SET LOCAL app.current_organization_id = orgID          │
│    • Execute SQL (parameterized)                            │
│    • Return domain model                                    │
│  })                                                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  PostgreSQL (RLS enforced)                                  │
│  • Policy: organization_id = current_setting(...)           │
│  • Only rows matching the tenant are visible                │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Audit Service (fire-and-forget for most modules)           │
│  • Persists event to audit_logs table                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  HTTP Response                                              │
│  Success: {"data": ...}                                     │
│  Error:   {"error": "message"}                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Module PR Review Checklist

Use this during pull request reviews for any module change:

### Database
- [ ] New tables have `organization_id TEXT NOT NULL`
- [ ] RLS enabled with `tenant_isolation_*` policy (USING + WITH CHECK)
- [ ] Migration file is sequentially numbered and never edits existing migrations
- [ ] Indexes added for organization_id and frequently-queried columns

### Repository
- [ ] All tenant-scoped queries use `database.WithTenantQuery`
- [ ] `organization_id` included in WHERE clauses (defense in depth)
- [ ] No raw pool queries for tenant data
- [ ] Sentinel errors returned (not raw pgx/database errors)

### Service
- [ ] Input validation (trim, required checks) in service, not handler
- [ ] `Store` interface defined for testability
- [ ] Audit event emitted for state-changing operations
- [ ] No HTTP concepts (status codes, headers, request/response types)

### Handler
- [ ] Thin: decode → call service → encode response
- [ ] Organization ID from `appcontext.GetOrganizationID(ctx)`
- [ ] Uses `pkg/response` helpers
- [ ] Domain errors mapped to appropriate HTTP status codes
- [ ] No business logic

### Architecture
- [ ] No cross-module imports (adapter interfaces in `routes.go` only)
- [ ] Routes registered in `registerRoutes()` with correct middleware/permissions
- [ ] New permissions added to role bootstrap if introduced
- [ ] Tests included (unit for service, integration for repository if applicable)
