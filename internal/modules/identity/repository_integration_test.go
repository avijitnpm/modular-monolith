package identity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationIdentityLifecycle(t *testing.T) {
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

	zitadelID := "zit-" + suffix
	email := "test-" + suffix + "@example.com"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM identities WHERE zitadel_user_id = $1", zitadelID)
	})

	// Lookup non-existent
	got, err := repo.GetByZitadelUserID(ctx, zitadelID)
	if err != nil {
		t.Fatalf("GetByZitadelUserID error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}

	// Create
	created, err := repo.Create(ctx, zitadelID, email, "TestUser")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ZitadelUserID != zitadelID || created.Email != email {
		t.Fatalf("unexpected identity: %+v", created)
	}

	// Lookup by zitadel user id
	got, err = repo.GetByZitadelUserID(ctx, zitadelID)
	if err != nil {
		t.Fatalf("GetByZitadelUserID error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	// Lookup by email
	got, err = repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	// Update
	newEmail := "updated-" + suffix + "@example.com"
	updated, err := repo.Update(ctx, zitadelID, newEmail, "UpdatedName")
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updated.Email != newEmail || updated.Name != "UpdatedName" {
		t.Fatalf("unexpected updated identity: %+v", updated)
	}
}

func TestRepositoryIntegrationUniqueConstraints(t *testing.T) {
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

	zitadelID1 := "zit-uc1-" + suffix
	zitadelID2 := "zit-uc2-" + suffix
	email := "uc-" + suffix + "@example.com"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM identities WHERE zitadel_user_id IN ($1, $2)", zitadelID1, zitadelID2)
	})

	_, err = repo.Create(ctx, zitadelID1, email, "First")
	if err != nil {
		t.Fatalf("Create first error: %v", err)
	}

	// Duplicate zitadel_user_id
	_, err = repo.Create(ctx, zitadelID1, "other-"+suffix+"@example.com", "Dup")
	if err == nil {
		t.Fatal("expected error for duplicate zitadel_user_id")
	}

	// Duplicate email
	_, err = repo.Create(ctx, zitadelID2, email, "Dup")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestRepositoryIntegrationGetMemberships(t *testing.T) {
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

	zitadelID := "zit-mem-" + suffix
	email := "mem-" + suffix + "@example.com"
	orgID := "org-mem-" + suffix

	// Create identity
	created, err := repo.Create(ctx, zitadelID, email, "MembershipTest")
	if err != nil {
		t.Fatalf("Create identity: %v", err)
	}

	// Create a user row with identity_id
	_, err = pool.Exec(ctx,
		`INSERT INTO users (organization_id, zitadel_user_id, email, identity_id)
		 VALUES ($1, $2, $3, $4)`,
		orgID, zitadelID, email, created.ID,
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE identity_id = $1", created.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM identities WHERE id = $1", created.ID)
	})

	memberships, err := repo.GetMemberships(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetMemberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].OrganizationID != orgID {
		t.Fatalf("expected org %s, got %s", orgID, memberships[0].OrganizationID)
	}
}
