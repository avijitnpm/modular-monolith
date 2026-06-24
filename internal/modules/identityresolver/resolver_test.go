package identityresolver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityResolver_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	zitadelID := "zit-res-" + suffix
	email := "res-" + suffix + "@example.com"

	_, err = pool.Exec(ctx,
		`INSERT INTO identities (zitadel_user_id, email, name) VALUES ($1, $2, 'ResolverTest')`,
		zitadelID, email)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM identities WHERE zitadel_user_id = $1", zitadelID)
	})

	resolver := NewIdentityResolver(pool)

	// Resolve existing
	id, err := resolver.ResolveIdentity(ctx, zitadelID)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.ProviderID != zitadelID || id.Email != email {
		t.Fatalf("unexpected identity: %+v", id)
	}

	// Resolve missing
	_, err = resolver.ResolveIdentity(ctx, "nonexistent")
	if err != ErrIdentityNotFound {
		t.Fatalf("expected ErrIdentityNotFound, got %v", err)
	}
}

func TestMembershipResolver_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	zitadelID := "zit-mres-" + suffix
	email := "mres-" + suffix + "@example.com"
	orgID := "org-mres-" + suffix

	// Create identity
	var identityID string
	err = pool.QueryRow(ctx,
		`INSERT INTO identities (zitadel_user_id, email, name) VALUES ($1, $2, 'MResTest') RETURNING id`,
		zitadelID, email).Scan(&identityID)
	if err != nil {
		t.Fatalf("setup identity: %v", err)
	}

	// Create user
	_, err = pool.Exec(ctx,
		`INSERT INTO users (organization_id, zitadel_user_id, email, identity_id) VALUES ($1, $2, $3, $4)`,
		orgID, zitadelID, email, identityID)
	if err != nil {
		t.Fatalf("setup user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE identity_id = $1", identityID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM identities WHERE id = $1", identityID)
	})

	resolver := NewMembershipResolver(pool)

	// Resolve existing
	m, err := resolver.ResolveMembership(ctx, identityID)
	if err != nil {
		t.Fatalf("ResolveMembership: %v", err)
	}
	if m.OrganizationID != orgID {
		t.Fatalf("expected org %s, got %s", orgID, m.OrganizationID)
	}

	// Resolve missing
	_, err = resolver.ResolveMembership(ctx, "00000000-0000-0000-0000-000000000000")
	if err != ErrMembershipNotFound {
		t.Fatalf("expected ErrMembershipNotFound, got %v", err)
	}
}
