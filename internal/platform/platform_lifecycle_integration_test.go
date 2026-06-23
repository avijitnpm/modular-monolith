package platform

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/modules/billing"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
	"github.com/avijitnpm/modular-monolith/internal/modules/rbac"
	"github.com/avijitnpm/modular-monolith/internal/modules/usage"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPlatformLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	orgA := "org-lifecycle-a-" + suffix
	orgB := "org-lifecycle-b-" + suffix

	t.Cleanup(func() {
		for _, id := range []string{orgA, orgB} {
			_, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM usage_counters WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM subscriptions WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM user_roles WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM role_permissions WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM roles WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE organization_id = $1", id)
			_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE organization_id = $1", id)
		}
	})

	// Services
	repo := repository.New(pool)
	auditService := audit.NewService(repo)
	svc := service.New(repo, auditService)
	billingRepo := billing.NewRepository(pool)
	billingService := billing.NewService(billingRepo, nil, auditService)
	rbacRepo := rbac.NewRepository(pool)
	rbacService := rbac.NewService(rbacRepo, auditService)
	usageRepo := usage.NewRepository(pool)
	usageAdapter := usage.NewAdapter(usageRepo)

	// === 1. Organization Bootstrap ===
	t.Run("organization_bootstrap", func(t *testing.T) {
		orgAResult, err := svc.RegisterOrganization(ctx, orgA, "Org A")
		if err != nil {
			t.Fatalf("RegisterOrganization A: %v", err)
		}
		if orgAResult.OrganizationID != orgA {
			t.Fatalf("expected %q, got %q", orgA, orgAResult.OrganizationID)
		}

		_, err = svc.RegisterOrganization(ctx, orgB, "Org B")
		if err != nil {
			t.Fatalf("RegisterOrganization B: %v", err)
		}

		// Verify default roles created
		var roleCount int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM roles WHERE organization_id = $1", orgA).Scan(&roleCount)
		if roleCount != 4 {
			t.Fatalf("expected 4 roles, got %d", roleCount)
		}

		// Verify audit log
		var auditCount int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE organization_id = $1 AND action = 'organization_created'", orgA).Scan(&auditCount)
		if auditCount != 1 {
			t.Fatalf("expected 1 audit log, got %d", auditCount)
		}
	})

	// === 2. User Lifecycle ===
	t.Run("user_lifecycle", func(t *testing.T) {
		userID := createUser(t, ctx, pool, orgA, "user-a-"+suffix, "a-"+suffix+"@test.com")

		// Create a role and assign
		role, err := rbacRepo.CreateRole(ctx, orgA, "tester-"+suffix, []string{"users.read", "billing.read"})
		if err != nil {
			t.Fatalf("CreateRole: %v", err)
		}

		_, err = rbacRepo.AssignRoleToUser(ctx, orgA, userID, role.ID)
		if err != nil {
			t.Fatalf("AssignRoleToUser: %v", err)
		}

		// Permission check
		has, err := rbacService.UserHasPermission(ctx, orgA, "user-a-"+suffix, "users.read")
		if err != nil {
			t.Fatalf("UserHasPermission: %v", err)
		}
		if !has {
			t.Fatal("expected user to have users.read")
		}
	})

	// === 3. Subscription Lifecycle ===
	t.Run("subscription_lifecycle", func(t *testing.T) {
		// Create trialing subscription
		sub, err := billingService.CreateSubscription(ctx, orgA, "manual", "pro", "trialing")
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Verify active (trialing counts)
		checker := billing.NewSubscriptionChecker(billingRepo)
		active, _ := checker.HasActiveSubscription(ctx, orgA)
		if !active {
			t.Fatal("expected trialing to be active")
		}

		// Update to active
		_, err = billingService.UpdateSubscription(ctx, sub.ID, orgA, "pro", "active", nil)
		if err != nil {
			t.Fatalf("UpdateSubscription active: %v", err)
		}
		active, _ = checker.HasActiveSubscription(ctx, orgA)
		if !active {
			t.Fatal("expected active subscription")
		}

		// Update to cancelled
		_, err = billingService.UpdateSubscription(ctx, sub.ID, orgA, "pro", "cancelled", nil)
		if err != nil {
			t.Fatalf("UpdateSubscription cancelled: %v", err)
		}
		active, _ = checker.HasActiveSubscription(ctx, orgA)
		if active {
			t.Fatal("expected cancelled to be inactive")
		}

		// Restore to active for later tests
		_, err = billingService.UpdateSubscription(ctx, sub.ID, orgA, "pro", "active", nil)
		if err != nil {
			t.Fatalf("UpdateSubscription restore: %v", err)
		}
	})

	// === 4. Usage Tracking ===
	t.Run("usage_tracking", func(t *testing.T) {
		_, err := usageRepo.IncrementUsage(ctx, orgA, "users", 3)
		if err != nil {
			t.Fatalf("IncrementUsage users: %v", err)
		}
		_, err = usageRepo.IncrementUsage(ctx, orgA, "documents", 50)
		if err != nil {
			t.Fatalf("IncrementUsage documents: %v", err)
		}
		_, err = usageRepo.IncrementUsage(ctx, orgA, "api_requests", 500)
		if err != nil {
			t.Fatalf("IncrementUsage api_requests: %v", err)
		}
		_, err = usageRepo.IncrementUsage(ctx, orgA, "storage", 1024)
		if err != nil {
			t.Fatalf("IncrementUsage storage: %v", err)
		}

		// Verify
		list, err := usageRepo.ListUsage(ctx, orgA)
		if err != nil {
			t.Fatalf("ListUsage: %v", err)
		}
		if len(list) != 4 {
			t.Fatalf("expected 4 counters, got %d", len(list))
		}
	})

	// === 5. Entitlements ===
	t.Run("entitlements", func(t *testing.T) {
		subAdapter := &testSubAdapter{store: billingRepo}

		// Pro plan (active subscription for orgA)
		entSvc := entitlements.NewService(subAdapter, usageAdapter)
		e, err := entSvc.CanUse(ctx, orgA, "users", 1)
		if err != nil {
			t.Fatalf("CanUse: %v", err)
		}
		if !e.Allowed {
			t.Fatal("expected allowed under pro limit")
		}
		if e.Limit != 10 {
			t.Fatalf("expected pro limit 10, got %d", e.Limit)
		}

		// Free plan (orgB has no subscription)
		eB, err := entSvc.CanUse(ctx, orgB, "users", 1)
		if err != nil {
			t.Fatalf("CanUse orgB: %v", err)
		}
		if eB.Limit != 1 {
			t.Fatalf("expected free limit 1, got %d", eB.Limit)
		}

		// Enterprise test via override
		entEnterprise := entitlements.NewService(
			&staticSubAdapter{plan: "enterprise", status: "active"},
			usageAdapter,
		)
		eEnt, err := entEnterprise.CanUse(ctx, orgA, "users", 1)
		if err != nil {
			t.Fatalf("CanUse enterprise: %v", err)
		}
		if eEnt.Limit != entitlements.Unlimited {
			t.Fatalf("expected unlimited, got %d", eEnt.Limit)
		}
		if !eEnt.Allowed {
			t.Fatal("expected enterprise always allowed")
		}
	})

	// === 6. Dashboard Consistency ===
	t.Run("dashboard_consistency", func(t *testing.T) {
		// Verify subscription matches
		sub, err := billingService.GetSubscription(ctx, orgA)
		if err != nil {
			t.Fatalf("GetSubscription: %v", err)
		}
		if sub == nil || sub.Plan != "pro" || sub.Status != "active" {
			t.Fatalf("expected active pro subscription, got %+v", sub)
		}

		// Verify usage matches
		users, _ := usageAdapter.GetUsage(ctx, orgA, "users")
		if users != 3 {
			t.Fatalf("expected users=3, got %d", users)
		}
		docs, _ := usageAdapter.GetUsage(ctx, orgA, "documents")
		if docs != 50 {
			t.Fatalf("expected documents=50, got %d", docs)
		}

		// Verify entitlements
		subAdapter := &testSubAdapter{store: billingRepo}
		entSvc := entitlements.NewService(subAdapter, usageAdapter)
		ents, err := entSvc.GetEntitlements(ctx, orgA)
		if err != nil {
			t.Fatalf("GetEntitlements: %v", err)
		}
		if len(ents) != 4 {
			t.Fatalf("expected 4 entitlements, got %d", len(ents))
		}
	})

	// === 7. Tenant Isolation ===
	t.Run("tenant_isolation", func(t *testing.T) {
		// Create data for orgB
		createUser(t, ctx, pool, orgB, "user-b-"+suffix, "b-"+suffix+"@test.com")
		_, _ = billingService.CreateSubscription(ctx, orgB, "manual", "free", "active")
		_, _ = usageRepo.IncrementUsage(ctx, orgB, "users", 1)

		// Subscriptions isolated
		subA, _ := billingRepo.GetSubscription(ctx, orgA)
		subB, _ := billingRepo.GetSubscription(ctx, orgB)
		if subA.Plan == subB.Plan && subA.Plan == "pro" {
			t.Fatal("subscription isolation issue: orgB should have free, not pro")
		}

		// Usage isolated
		usageA, _ := usageAdapter.GetUsage(ctx, orgA, "users")
		usageB, _ := usageAdapter.GetUsage(ctx, orgB, "users")
		if usageA == usageB {
			t.Fatal("usage isolation issue: orgA and orgB should have different user counts")
		}
		if usageA != 3 || usageB != 1 {
			t.Fatalf("expected usageA=3 usageB=1, got %d %d", usageA, usageB)
		}

		// Roles isolated
		rolesA, _ := rbacRepo.ListRoles(ctx, orgA)
		rolesB, _ := rbacRepo.ListRoles(ctx, orgB)
		for _, r := range rolesB {
			for _, ra := range rolesA {
				if r.ID == ra.ID {
					t.Fatal("role isolation broken: shared role ID across tenants")
				}
			}
		}

		// Audit logs isolated
		var auditCountA, auditCountB int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE organization_id = $1", orgA).Scan(&auditCountA)
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE organization_id = $1", orgB).Scan(&auditCountB)
		if auditCountA == 0 {
			t.Fatal("expected audit logs for orgA")
		}
		if auditCountB == 0 {
			t.Fatal("expected audit logs for orgB")
		}
	})
}

func createUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID, zitadelUserID, email string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO users (organization_id, zitadel_user_id, email) VALUES ($1, $2, $3) RETURNING id`,
		orgID, zitadelUserID, email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return id
}

type testSubAdapter struct {
	store billing.Store
}

func (a *testSubAdapter) GetSubscription(ctx context.Context, organizationID string) (string, string, error) {
	sub, err := a.store.GetSubscription(ctx, organizationID)
	if err != nil {
		return "", "", err
	}
	if sub == nil {
		return "", "", nil
	}
	return sub.Plan, sub.Status, nil
}

type staticSubAdapter struct {
	plan   string
	status string
}

func (a *staticSubAdapter) GetSubscription(ctx context.Context, organizationID string) (string, string, error) {
	return a.plan, a.status, nil
}
