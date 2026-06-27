# Engineering Rules

Permanent engineering rules for this repository. These are non-negotiable.

---

## Always

1. **Always use `database.WithTenantQuery`** for any query against tenant-scoped data.
2. **Always include `organization_id` in WHERE clauses** even though RLS enforces isolation. Defense in depth.
3. **Always enable RLS** on new tables that contain tenant-scoped data. Add both USING and WITH CHECK policies.
4. **Always emit audit events** for state-changing operations (create, update, delete, role changes, billing events).
5. **Always use dependency injection**. Services receive interfaces, not concrete types. Handlers receive services, not repositories.
6. **Always keep handlers thin**. Decode request → call service → encode response. No business logic in handlers.
7. **Always keep repositories persistence-only**. No business logic, no validation, no conditional branching beyond error mapping.
8. **Always validate in the service layer**. Trim strings, check required fields, enforce business invariants.
9. **Always use `pkg/response` helpers** for HTTP responses. Never write raw JSON encoding in handlers.
10. **Always return sentinel errors from services**. Handlers map them to HTTP status codes.
11. **Always define `Store` interfaces** in services for testability.
12. **Always use parameterized queries**. Never concatenate user input into SQL.
13. **Always run `gofmt`** before committing. CI rejects unformatted code.
14. **Always use structured logging** (`slog`). Never use `fmt.Println` or `log.Printf`.
15. **Always include `organization_id` as a column** in every tenant-scoped table.

---

## Never

1. **Never import another module's package directly**. Use adapter interfaces in `routes.go`.
2. **Never query the pool directly** for tenant-scoped data. This bypasses RLS entirely.
3. **Never disable tenant isolation**. There is no admin override. Cross-tenant operations must be explicitly designed and audited.
4. **Never put business logic in handlers**. If you're writing an `if` that isn't about HTTP status, it belongs in the service.
5. **Never expose raw database errors** to API clients. Map them to domain errors and return generic messages.
6. **Never store secrets in code**. All secrets come from environment variables or Docker secrets (`/run/secrets/`).
7. **Never bypass the service layer**. Handlers call services. Services call repositories. No shortcuts.
8. **Never add a global mutable variable**. All state flows through dependency injection.
9. **Never use `fmt.Sprintf` to build SQL**. Always use parameterized queries (`$1`, `$2`).
10. **Never commit `.env` files**. Only `.env.example` is committed.
11. **Never push directly to master**. All changes go through pull requests with CI passing.
12. **Never add an ORM**. Raw SQL with pgx is intentional. ORMs hide query behavior.

---

## Architecture Rules

1. **Package dependency direction is strictly downward**:
   ```
   cmd → internal/app → internal/router → internal/modules
   internal/modules → internal/database, internal/context, pkg/
   pkg/ → standard library only
   ```

2. **Modules are isolated units**. A module contains its handler, service, repository, models, and errors. It exposes only what `routes.go` needs.

3. **The router is the composition root**. All wiring (instantiation, dependency injection, adapter creation) happens in `internal/router/routes.go`. Nowhere else.

4. **Frontend is a build artifact**. The Go application serves it, but it is architecturally separate. The frontend never calls internal Go packages — it communicates only via the HTTP API.

5. **Migrations are append-only**. Never edit an existing migration file after it has been applied. Always create a new migration.

---

## Dependency Rules

1. **Direct dependencies require justification**. Adding a new dependency to `go.mod` must solve a real problem that the stdlib cannot handle reasonably.
2. **Pin dependency versions exactly**. No floating version ranges.
3. **No duplicate functionality**. If `pkg/validator` does email validation, don't add a third-party validation library.
4. **`internal/` packages cannot import `cmd/`**. Direction is always inward.

---

## Testing Rules

1. **Unit tests for services**. Mock the `Store` interface and test business logic.
2. **Integration tests for repositories**. Use a real PostgreSQL instance (testcontainers or test database).
3. **Short tests by default**. `go test ./... -short` must pass without external dependencies (database, network).
4. **Race detection in CI**. All tests run with `-race`.
5. **Tests must not depend on execution order**. Each test sets up its own state.
6. **Name test files `*_test.go`** in the same package. Use `_integration_test.go` suffix for tests requiring external services.

---

## Security Rules

1. **RLS is the primary tenant isolation mechanism**. Application-level filtering is secondary defense.
2. **Session secrets must be at least 32 random characters** in production.
3. **Never log PII** (emails, tokens, passwords) at INFO level. Use the logger's redaction layer.
4. **Rate limiting is mandatory** on public endpoints (login, callback, onboarding, webhooks).
5. **Webhook endpoints must verify signatures** before processing payloads.
6. **Security headers** (HSTS, CSP, X-Frame-Options) are enforced at the Traefik layer.
7. **Containers run as non-root** (UID 10001) with `no-new-privileges`.
8. **The app container is read-only**. Only `/tmp` is writable (via tmpfs).

---

## Performance Rules

1. **No N+1 queries**. Use JOINs or batch queries. Pagination is mandatory for list endpoints.
2. **Connection pool limits** are set in PostgreSQL configuration, not overridden in application code.
3. **HTTP server timeouts are explicit**: ReadHeaderTimeout (5s), ReadTimeout (15s), WriteTimeout (30s), IdleTimeout (60s).
4. **Body size limits** are enforced globally (1MB) via middleware.
5. **OTEL tracing is async**. If the backend is down, spans are dropped — never block on observability.

---

## Module Rules

1. **One module = one bounded context**. Don't combine unrelated responsibilities.
2. **Modules own their tables**. No two modules write to the same table.
3. **Cross-module communication via interfaces only**. Define adapters in `routes.go`.
4. **New permissions must be added to role bootstrap**. Otherwise new organizations won't have roles that include the new permission.
5. **Modules must not assume request context**. Always extract from `appcontext`, never from headers or URL path (that's the handler's job).


---

## Documented Exceptions

Intentional exceptions to the rules above. Each exists for a specific reason.

| Exception | Rule Violated | Why It Exists |
|-----------|---------------|---------------|
| `identityresolver` queries the pool directly (no `WithTenantQuery`) | Always use WithTenantQuery | Identity resolution must work across tenants. An identity maps to memberships in multiple organizations. RLS would filter to a single tenant, making cross-org lookup impossible. |
| `permissions` table has no RLS | Always enable RLS on tenant data | Permissions are global system definitions (e.g., "billing.read"). They are not tenant-scoped. Every organization shares the same permission set. |
| `identities` table has no RLS | Always enable RLS on tenant data | Identities represent external provider users (Zitadel subjects). They exist before any organization membership and can belong to multiple organizations. |
| `schema_migrations` table has no RLS | Always enable RLS on tenant data | Migration tracking is infrastructure, not tenant data. It has no `organization_id`. |
| `internal/repository/` (shared repository) | Modules own their tables | Legacy package predating the module system. Owns `users`, `organizations`, `audit_logs`. These tables were created before per-module ownership was established. New modules must not add to this package. |
| `internal/service/` (shared service) | Modules own their services | Wraps the shared repository. Same legacy reason. |
| `health` module has no service/repository | Module structure requires handler+service+repo | Health check is a single function that pings the database. No business logic or persistence exists. Adding service/repo layers would be empty ceremony. |
| `entitlements` module has no repository | Module structure requires handler+service+repo | Entitlements are computed from billing (subscription plan) + usage (current counts). It reads from other modules via interfaces — it owns no data. |
| Migration runner queries without `WithTenantQuery` | Always use WithTenantQuery | Migrations run at startup before any tenant context exists. They operate on DDL (schema changes), which are global by nature. |
| `onboarding` and `invitations` read session cookies directly | Protected routes use middleware chain | These endpoints are semi-public — they authenticate the user but cannot resolve membership (user has no membership yet). They read the session directly instead of going through the full SessionIdentity → ResolveMembership → TenantContext chain. |
