package service

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRegisterOrganizationBootstrapsRBAC(t *testing.T) {
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

	svc := New(
		repository.New(pool),
		nil,
	)

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	zitadelOrgID := "org-register-" + suffix

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM role_permissions WHERE organization_id = $1",
			zitadelOrgID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM roles WHERE organization_id = $1",
			zitadelOrgID,
		)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM organizations WHERE organization_id = $1",
			zitadelOrgID,
		)
	})

	org, err := svc.RegisterOrganization(
		ctx,
		zitadelOrgID,
		"Registered Org",
	)

	if err != nil {
		t.Fatalf("RegisterOrganization returned error: %v", err)
	}

	if org.OrganizationID != zitadelOrgID {
		t.Fatalf("expected organization_id %q, got %q", zitadelOrgID, org.OrganizationID)
	}

	var roleCount int

	err = pool.QueryRow(
		ctx,
		"SELECT count(*) FROM roles WHERE organization_id = $1",
		zitadelOrgID,
	).Scan(
		&roleCount,
	)

	if err != nil {
		t.Fatalf("failed to count roles: %v", err)
	}

	if roleCount != 4 {
		t.Fatalf("expected four default roles, got %d", roleCount)
	}

	rows, err := pool.Query(
		ctx,
		"SELECT name FROM roles WHERE organization_id = $1 ORDER BY name",
		zitadelOrgID,
	)

	if err != nil {
		t.Fatalf("failed to query roles: %v", err)
	}

	defer rows.Close()

	roleNames := []string{}

	for rows.Next() {
		var roleName string

		err = rows.Scan(
			&roleName,
		)

		if err != nil {
			t.Fatalf("failed to scan role name: %v", err)
		}

		roleNames = append(
			roleNames,
			roleName,
		)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to read role rows: %v", err)
	}

	expectedRoleNames := []string{
		"admin",
		"member",
		"owner",
		"viewer",
	}

	sort.Strings(roleNames)

	if len(roleNames) != len(expectedRoleNames) {
		t.Fatalf("expected role names %#v, got %#v", expectedRoleNames, roleNames)
	}

	for i, expectedRoleName := range expectedRoleNames {
		if roleNames[i] != expectedRoleName {
			t.Fatalf("expected role names %#v, got %#v", expectedRoleNames, roleNames)
		}
	}

	var rolePermissionCount int

	err = pool.QueryRow(
		ctx,
		"SELECT count(*) FROM role_permissions WHERE organization_id = $1",
		zitadelOrgID,
	).Scan(
		&rolePermissionCount,
	)

	if err != nil {
		t.Fatalf("failed to count role permissions: %v", err)
	}

	if rolePermissionCount != 19 {
		t.Fatalf("expected 19 role permissions, got %d", rolePermissionCount)
	}
}
