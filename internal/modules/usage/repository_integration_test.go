package usage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationUsageLifecycle(t *testing.T) {
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

	repo := NewRepository(pool)
	suffix := time.Now().UTC().Format("20060102150405.000000000")
	orgID := "org-usage-" + suffix

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM usage_counters WHERE organization_id = $1", orgID)
	})

	// Get non-existent
	got, err := repo.GetUsage(ctx, orgID, "users")
	if err != nil {
		t.Fatalf("GetUsage error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}

	// Increment creates
	counter, err := repo.IncrementUsage(ctx, orgID, "users", 1)
	if err != nil {
		t.Fatalf("IncrementUsage error: %v", err)
	}
	if counter.Value != 1 {
		t.Fatalf("expected 1, got %d", counter.Value)
	}

	// Increment adds
	counter, err = repo.IncrementUsage(ctx, orgID, "users", 5)
	if err != nil {
		t.Fatalf("IncrementUsage error: %v", err)
	}
	if counter.Value != 6 {
		t.Fatalf("expected 6, got %d", counter.Value)
	}

	// SetUsage overwrites
	counter, err = repo.SetUsage(ctx, orgID, "users", 100)
	if err != nil {
		t.Fatalf("SetUsage error: %v", err)
	}
	if counter.Value != 100 {
		t.Fatalf("expected 100, got %d", counter.Value)
	}

	// Add another metric
	_, err = repo.IncrementUsage(ctx, orgID, "documents", 3)
	if err != nil {
		t.Fatalf("IncrementUsage documents error: %v", err)
	}

	// ListUsage
	list, err := repo.ListUsage(ctx, orgID)
	if err != nil {
		t.Fatalf("ListUsage error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 counters, got %d", len(list))
	}
}

func TestRepositoryIntegrationTenantIsolation(t *testing.T) {
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

	repo := NewRepository(pool)
	suffix := time.Now().UTC().Format("20060102150405.000000000")
	orgA := "org-usage-a-" + suffix
	orgB := "org-usage-b-" + suffix

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM usage_counters WHERE organization_id IN ($1, $2)", orgA, orgB)
	})

	// Create usage for org A
	_, err = repo.IncrementUsage(ctx, orgA, "users", 10)
	if err != nil {
		t.Fatalf("IncrementUsage orgA error: %v", err)
	}

	// Create usage for org B
	_, err = repo.IncrementUsage(ctx, orgB, "users", 20)
	if err != nil {
		t.Fatalf("IncrementUsage orgB error: %v", err)
	}

	// Org A cannot see org B's usage
	gotA, err := repo.GetUsage(ctx, orgA, "users")
	if err != nil {
		t.Fatalf("GetUsage orgA error: %v", err)
	}
	if gotA.Value != 10 {
		t.Fatalf("orgA expected 10, got %d", gotA.Value)
	}

	// Org B cannot see org A's usage
	gotB, err := repo.GetUsage(ctx, orgB, "users")
	if err != nil {
		t.Fatalf("GetUsage orgB error: %v", err)
	}
	if gotB.Value != 20 {
		t.Fatalf("orgB expected 20, got %d", gotB.Value)
	}

	// List for org A only returns org A
	listA, err := repo.ListUsage(ctx, orgA)
	if err != nil {
		t.Fatalf("ListUsage orgA error: %v", err)
	}
	for _, c := range listA {
		if c.OrganizationID != orgA {
			t.Fatalf("tenant isolation broken: orgA list contains %s", c.OrganizationID)
		}
	}
}
