package service

import (
	"context"
	"os"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeAuditLogger struct {
	events []*audit.Event
}

func (f *fakeAuditLogger) Log(_ context.Context, event *audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func TestRegisterOrganizationEmitsAuditEvent(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)

	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	defer pool.Close()

	auditLog := &fakeAuditLogger{}
	svc := New(repository.New(pool), auditLog)

	org, err := svc.RegisterOrganization(
		ctx,
		"audit-test-org-1",
		"Audit Test Org",
	)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM role_permissions WHERE organization_id = $1", "audit-test-org-1")
		_, _ = pool.Exec(context.Background(), "DELETE FROM roles WHERE organization_id = $1", "audit-test-org-1")
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE organization_id = $1", "audit-test-org-1")
		_, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE organization_id = $1", "audit-test-org-1")
	})

	if err != nil {
		t.Fatalf("RegisterOrganization returned error: %v", err)
	}

	if len(auditLog.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditLog.events))
	}

	e := auditLog.events[0]

	if e.Action != "organization_created" {
		t.Errorf("expected action organization_created, got %q", e.Action)
	}

	if e.EntityType != "organization" {
		t.Errorf("expected entity_type organization, got %q", e.EntityType)
	}

	if e.EntityID != org.OrganizationID {
		t.Errorf("expected entity_id %q, got %q", org.OrganizationID, e.EntityID)
	}

	if e.OrganizationID != org.OrganizationID {
		t.Errorf("expected organization_id %q, got %q", org.OrganizationID, e.OrganizationID)
	}

	if e.Metadata == nil {
		t.Fatal("expected metadata to be populated")
	}

	if e.Metadata["name"] != "Audit Test Org" {
		t.Errorf("expected metadata name=Audit Test Org, got %q", e.Metadata["name"])
	}

	if e.Metadata["zitadel_org_id"] != "audit-test-org-1" {
		t.Errorf("expected metadata zitadel_org_id=audit-test-org-1, got %q", e.Metadata["zitadel_org_id"])
	}
}

func TestRegisterOrganizationNilAuditDoesNotPanic(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)

	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	defer pool.Close()

	svc := New(repository.New(pool), nil)

	_, err = svc.RegisterOrganization(
		ctx,
		"audit-nil-test-org",
		"Nil Audit Org",
	)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM role_permissions WHERE organization_id = $1", "audit-nil-test-org")
		_, _ = pool.Exec(context.Background(), "DELETE FROM roles WHERE organization_id = $1", "audit-nil-test-org")
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE organization_id = $1", "audit-nil-test-org")
	})

	if err != nil {
		t.Fatalf("RegisterOrganization with nil audit should not error: %v", err)
	}
}
