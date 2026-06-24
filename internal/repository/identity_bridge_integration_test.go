package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avijitnpm/modular-monolith/internal/database"
)

func TestGetByIdentityID_Integration(t *testing.T) {
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
	orgID := "org-bridge-" + suffix
	zitadelID := "zit-bridge-" + suffix
	email := "bridge-" + suffix + "@example.com"

	// Create identity first
	var identityID string
	err = pool.QueryRow(ctx,
		`INSERT INTO identities (zitadel_user_id, email, name)
		 VALUES ($1, $2, 'BridgeTest')
		 RETURNING id`, zitadelID, email,
	).Scan(&identityID)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE zitadel_user_id = $1", zitadelID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM identities WHERE id = $1", identityID)
	})

	// Create user with identity_id via the repository
	repo := &Repository{DB: pool}
	user, err := repo.CreateUser(ctx, orgID, zitadelID, email)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.IdentityID != identityID {
		t.Fatalf("expected identity_id %s, got %s", identityID, user.IdentityID)
	}

	// GetByIdentityID
	got, err := repo.GetByIdentityID(ctx, orgID, identityID)
	if err != nil {
		t.Fatalf("GetByIdentityID: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, got.ID)
	}
	if got.IdentityID != identityID {
		t.Fatalf("expected identity_id %s, got %s", identityID, got.IdentityID)
	}
}

func TestListByIdentityID_Integration(t *testing.T) {
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
	orgID := "org-list-" + suffix
	zitadelID := "zit-list-" + suffix
	email := "list-" + suffix + "@example.com"

	// Create identity
	var identityID string
	err = pool.QueryRow(ctx,
		`INSERT INTO identities (zitadel_user_id, email, name)
		 VALUES ($1, $2, 'ListTest')
		 RETURNING id`, zitadelID, email,
	).Scan(&identityID)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	// Create user via direct INSERT (bypasses UNIQUE constraint issue for single-org test)
	var userID string
	err = database.WithTenantQuery(pool, ctx, orgID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO users (organization_id, zitadel_user_id, email, identity_id)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			orgID, zitadelID, email, identityID,
		).Scan(&userID)
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE identity_id = $1", identityID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM identities WHERE id = $1", identityID)
	})

	// ListByIdentityID (cross-org, no RLS)
	repo := &Repository{DB: pool}
	users, err := repo.ListByIdentityID(ctx, identityID)
	if err != nil {
		t.Fatalf("ListByIdentityID: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].ID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, users[0].ID)
	}
	if users[0].IdentityID != identityID {
		t.Fatalf("expected identity_id %s, got %s", identityID, users[0].IdentityID)
	}
}
