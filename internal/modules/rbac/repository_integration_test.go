package rbac

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationRBACFlow(t *testing.T) {
	databaseURL := os.Getenv(
		"TEST_DATABASE_URL",
	)

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(
		ctx,
		databaseURL,
	)

	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	defer pool.Close()

	repository := NewRepository(
		pool,
	)

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	organizationID := "org-rbac-" + suffix
	otherOrganizationID := "org-rbac-other-" + suffix
	zitadelUserID := "zitadel-rbac-" + suffix

	userID := createIntegrationUser(
		t,
		ctx,
		pool,
		organizationID,
		zitadelUserID,
		"rbac-"+suffix+"@example.com",
	)

	otherUserID := createIntegrationUser(
		t,
		ctx,
		pool,
		otherOrganizationID,
		"zitadel-rbac-other-"+suffix,
		"rbac-other-"+suffix+"@example.com",
	)

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM user_roles WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM role_permissions WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM roles WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM users WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM organizations WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
	})

	permissions, err := repository.ListPermissions(
		ctx,
	)

	if err != nil {
		t.Fatalf("ListPermissions returned error: %v", err)
	}

	if len(permissions) == 0 {
		t.Fatal("expected seeded permissions")
	}

	role, err := repository.CreateRole(
		ctx,
		organizationID,
		"support-"+suffix,
		[]string{"users.read", "settings.read"},
	)

	if err != nil {
		t.Fatalf("CreateRole returned error: %v", err)
	}

	if len(role.Permissions) != 2 {
		t.Fatalf("expected two permissions, got %#v", role.Permissions)
	}

	roles, err := repository.ListRoles(
		ctx,
		organizationID,
	)

	if err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}

	if len(roles) != 1 {
		t.Fatalf("expected one role for tenant, got %d", len(roles))
	}

	otherRoles, err := repository.ListRoles(
		ctx,
		otherOrganizationID,
	)

	if err != nil {
		t.Fatalf("ListRoles for other tenant returned error: %v", err)
	}

	if len(otherRoles) != 0 {
		t.Fatalf("expected tenant isolation, got roles %#v", otherRoles)
	}

	assignment, err := repository.AssignRoleToUser(
		ctx,
		organizationID,
		userID,
		role.ID,
	)

	if err != nil {
		t.Fatalf("AssignRoleToUser returned error: %v", err)
	}

	if assignment.UserID != userID {
		t.Fatalf("expected assignment for user %q, got %q", userID, assignment.UserID)
	}

	hasPermission, err := repository.UserHasPermission(
		ctx,
		organizationID,
		zitadelUserID,
		"users.read",
	)

	if err != nil {
		t.Fatalf("UserHasPermission returned error: %v", err)
	}

	if !hasPermission {
		t.Fatal("expected user to have users.read")
	}

	hasOtherTenantPermission, err := repository.UserHasPermission(
		ctx,
		otherOrganizationID,
		zitadelUserID,
		"users.read",
	)

	if err != nil {
		t.Fatalf("UserHasPermission for other tenant returned error: %v", err)
	}

	if hasOtherTenantPermission {
		t.Fatal("expected permission check to be tenant isolated")
	}

	_, err = repository.AssignRoleToUser(
		ctx,
		otherOrganizationID,
		otherUserID,
		role.ID,
	)

	if err == nil {
		t.Fatal("expected cross-tenant role assignment to fail")
	}

	removed, err := repository.RemoveRoleFromUser(
		ctx,
		organizationID,
		userID,
		role.ID,
	)

	if err != nil {
		t.Fatalf("RemoveRoleFromUser returned error: %v", err)
	}

	if removed.ID != assignment.ID {
		t.Fatalf("expected removed assignment %q, got %q", assignment.ID, removed.ID)
	}
}

func TestRepositoryIntegrationBootstrapDefaultRoles(t *testing.T) {
	databaseURL := os.Getenv(
		"TEST_DATABASE_URL",
	)

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(
		ctx,
		databaseURL,
	)

	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	defer pool.Close()

	repository := NewRepository(
		pool,
	)

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	organizationID := "org-rbac-bootstrap-" + suffix

	createIntegrationOrganization(
		t,
		ctx,
		pool,
		organizationID,
	)

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM role_permissions WHERE organization_id = $1",
			organizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM roles WHERE organization_id = $1",
			organizationID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM organizations WHERE organization_id = $1",
			organizationID,
		)
	})

	for i := 0; i < 2; i++ {
		err = repository.BootstrapDefaultRoles(
			ctx,
			organizationID,
		)

		if err != nil {
			t.Fatalf("BootstrapDefaultRoles returned error: %v", err)
		}
	}

	rolePermissions := map[string]int{}

	rows, err := pool.Query(
		ctx,
		`
			SELECT r.name, count(rp.permission_id)
			FROM roles r
			LEFT JOIN role_permissions rp
				ON rp.role_id = r.id
				AND rp.organization_id = r.organization_id
			WHERE r.organization_id = $1
			GROUP BY r.name
		`,
		organizationID,
	)

	if err != nil {
		t.Fatalf("failed to query bootstrapped roles: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var roleName string
		var permissionCount int

		err = rows.Scan(
			&roleName,
			&permissionCount,
		)

		if err != nil {
			t.Fatalf("failed to scan role permission count: %v", err)
		}

		rolePermissions[roleName] = permissionCount
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to read role permission counts: %v", err)
	}

	expected := map[string]int{
		"owner":  7,
		"admin":  6,
		"member": 2,
		"viewer": 4,
	}

	for roleName, permissionCount := range expected {
		if rolePermissions[roleName] != permissionCount {
			t.Fatalf(
				"expected %s to have %d permissions, got %d in %#v",
				roleName,
				permissionCount,
				rolePermissions[roleName],
				rolePermissions,
			)
		}
	}
}

func createIntegrationUser(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	organizationID string,
	zitadelUserID string,
	email string,
) string {

	t.Helper()

	createIntegrationOrganization(
		t,
		ctx,
		pool,
		organizationID,
	)

	var userID string

	err := pool.QueryRow(
		ctx,
		`
			INSERT INTO users (organization_id, zitadel_user_id, email)
			VALUES ($1, $2, $3)
			RETURNING id
		`,
		organizationID,
		zitadelUserID,
		email,
	).Scan(
		&userID,
	)

	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return userID
}

func createIntegrationOrganization(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	organizationID string,
) {

	t.Helper()

	_, err := pool.Exec(
		ctx,
		`
			INSERT INTO organizations (zitadel_org_id, name, organization_id)
			VALUES ($1, $2, $3)
		`,
		fmt.Sprintf("zitadel-%s", organizationID),
		organizationID,
		organizationID,
	)

	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}
}
