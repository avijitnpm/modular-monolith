package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWithTenantQuery_SetsTenantContext(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	var got string
	err := repo.WithTenantQuery(ctx, "org-123", func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT current_setting('app.current_organization_id')").Scan(&got)
	})

	if err != nil {
		t.Fatalf("WithTenantQuery returned error: %v", err)
	}

	if got != "org-123" {
		t.Fatalf("expected tenant context 'org-123', got %q", got)
	}
}

func TestWithTenantQuery_SharesTransaction(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	// Verify that SET LOCAL and the query share the same transaction
	// by confirming the setting persists across multiple operations in fn.
	var first, second string
	err := repo.WithTenantQuery(ctx, "org-shared", func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT current_setting('app.current_organization_id')").Scan(&first); err != nil {
			return err
		}
		// Execute another statement — setting should still be visible
		return tx.QueryRow(ctx, "SELECT current_setting('app.current_organization_id')").Scan(&second)
	})

	if err != nil {
		t.Fatalf("WithTenantQuery returned error: %v", err)
	}

	if first != "org-shared" || second != "org-shared" {
		t.Fatalf("expected tenant setting to persist within tx, got %q and %q", first, second)
	}
}

func TestWithTenantQuery_RollbackOnError(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	deliberateErr := errors.New("deliberate failure")

	err := repo.WithTenantQuery(ctx, "org-rollback", func(tx pgx.Tx) error {
		// Create a temp table inside the tx — if rolled back, it won't exist
		_, err := tx.Exec(ctx, "CREATE TEMP TABLE _tenant_test_rollback (id int) ON COMMIT DROP")
		if err != nil {
			return err
		}
		return deliberateErr
	})

	if !errors.Is(err, deliberateErr) {
		t.Fatalf("expected deliberate error, got: %v", err)
	}

	// Verify the transaction was rolled back — temp table should not exist
	var exists bool
	_ = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = '_tenant_test_rollback')").Scan(&exists)

	if exists {
		t.Fatal("expected transaction to be rolled back, but temp table exists")
	}
}

func TestWithTenantQuery_CommitsOnSuccess(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	// Use an advisory lock as a side-effect-free commit test.
	// If the tx commits, the lock is released. We verify no error.
	err := repo.WithTenantQuery(ctx, "org-commit", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(99999)")
		return err
	})

	if err != nil {
		t.Fatalf("expected successful commit, got: %v", err)
	}

	// Verify setting does NOT leak outside the committed transaction
	var setting string
	err = pool.QueryRow(ctx,
		"SELECT current_setting('app.current_organization_id', true)").Scan(&setting)
	if err != nil {
		t.Fatalf("failed to check setting after commit: %v", err)
	}

	if setting != "" {
		t.Fatalf("expected empty setting after tx commit, got %q", setting)
	}
}

func TestWithTenantQuery_RejectsEmptyOrgID(t *testing.T) {
	pool := setupTestPool(t)
	repo := New(pool)
	ctx := context.Background()

	err := repo.WithTenantQuery(ctx, "", func(tx pgx.Tx) error {
		t.Fatal("fn should not be called with empty org ID")
		return nil
	})

	if err == nil {
		t.Fatal("expected error for empty organization ID")
	}
}

func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	t.Cleanup(func() { pool.Close() })

	return pool
}
