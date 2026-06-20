package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestAuditLogRLS_TenantIsolation(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	orgA := "org-rls-audit-a"
	orgB := "org-rls-audit-b"

	// Insert audit log for org A (bypass RLS using superuser connection)
	_, err := pool.Exec(ctx,
		`INSERT INTO audit_logs (organization_id, action, entity_type) VALUES ($1, 'test_action', 'test_entity')`,
		orgA,
	)
	if err != nil {
		t.Fatalf("insert org A audit log: %v", err)
	}

	// Insert audit log for org B
	_, err = pool.Exec(ctx,
		`INSERT INTO audit_logs (organization_id, action, entity_type) VALUES ($1, 'test_action', 'test_entity')`,
		orgB,
	)
	if err != nil {
		t.Fatalf("insert org B audit log: %v", err)
	}

	// Query as org A — should only see org A's logs
	var count int
	err = repo.WithTenantQuery(ctx, orgA, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs WHERE organization_id = $1`, orgB,
		).Scan(&count)
	})
	if err != nil {
		t.Fatalf("query as org A: %v", err)
	}

	if count != 0 {
		t.Fatalf("org A can see %d of org B's audit logs, want 0", count)
	}

	// Cleanup
	pool.Exec(ctx, `DELETE FROM audit_logs WHERE organization_id IN ($1, $2)`, orgA, orgB)
}
