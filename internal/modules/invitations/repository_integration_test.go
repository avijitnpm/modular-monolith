package invitations

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationInvitationLifecycle(t *testing.T) {
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
	orgID := "org-inv-" + suffix
	email := "invite-" + suffix + "@example.com"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM invitations WHERE organization_id = $1", orgID)
	})

	// Create invitation
	inv, err := repo.Create(ctx, orgID, email, "member", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if inv.Token == "" {
		t.Fatal("expected token")
	}
	if inv.Email != email {
		t.Fatalf("expected email %s, got %s", email, inv.Email)
	}

	// Get by token
	got, err := repo.GetByToken(ctx, inv.Token)
	if err != nil {
		t.Fatalf("GetByToken error: %v", err)
	}
	if got.ID != inv.ID {
		t.Fatalf("expected ID %s, got %s", inv.ID, got.ID)
	}

	// Mark accepted
	err = repo.MarkAccepted(ctx, inv.Token)
	if err != nil {
		t.Fatalf("MarkAccepted error: %v", err)
	}

	got, err = repo.GetByToken(ctx, inv.Token)
	if err != nil {
		t.Fatalf("GetByToken after accept error: %v", err)
	}
	if got.AcceptedAt == nil {
		t.Fatal("expected accepted_at to be set")
	}

	// Non-existent token
	got, err = repo.GetByToken(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByToken nonexistent error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent token")
	}
}
