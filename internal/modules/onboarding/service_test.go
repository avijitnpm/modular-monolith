package onboarding

import (
	"context"
	"errors"
	"testing"
)

type mockOrgCreator struct {
	orgID string
	err   error
}

func (m *mockOrgCreator) RegisterOrganization(_ context.Context, orgID, name string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	m.orgID = orgID
	return orgID, nil
}

type mockUserCreator struct {
	userID string
	err    error
}

func (m *mockUserCreator) CreateUser(_ context.Context, organizationID, zitadelUserID, email string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.userID == "" {
		m.userID = "user-1"
	}
	return m.userID, nil
}

type mockRoleAssigner struct {
	called bool
	err    error
}

func (m *mockRoleAssigner) AssignOwnerRole(_ context.Context, organizationID, userID string) error {
	m.called = true
	return m.err
}

type mockAuditLogger struct {
	called bool
}

func (m *mockAuditLogger) LogOnboarding(_ context.Context, organizationID, userID string, metadata map[string]string) error {
	m.called = true
	return nil
}

type mockIdentityChecker struct {
	has bool
	err error
}

func (m *mockIdentityChecker) HasOrganization(_ context.Context, zitadelUserID string) (bool, error) {
	return m.has, m.err
}

func TestCompleteOnboarding_Success(t *testing.T) {
	roles := &mockRoleAssigner{}
	audit := &mockAuditLogger{}
	svc := NewService(
		&mockOrgCreator{},
		&mockUserCreator{userID: "user-1"},
		roles,
		audit,
		&mockIdentityChecker{has: false},
	)

	result, err := svc.CompleteOnboarding(context.Background(), "zit-1", "a@b.com", "Acme Inc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OrganizationID == "" {
		t.Fatal("expected organization ID")
	}
	if result.OrganizationName != "Acme Inc" {
		t.Fatalf("expected org name Acme Inc, got %s", result.OrganizationName)
	}
	if result.UserID != "user-1" {
		t.Fatalf("expected user ID user-1, got %s", result.UserID)
	}
	if !roles.called {
		t.Fatal("expected owner role assignment")
	}
	if !audit.called {
		t.Fatal("expected audit log")
	}
}

func TestCompleteOnboarding_AlreadyOnboarded(t *testing.T) {
	svc := NewService(
		&mockOrgCreator{},
		&mockUserCreator{},
		&mockRoleAssigner{},
		&mockAuditLogger{},
		&mockIdentityChecker{has: true},
	)

	_, err := svc.CompleteOnboarding(context.Background(), "zit-1", "a@b.com", "Acme")
	if !errors.Is(err, ErrAlreadyOnboarded) {
		t.Fatalf("expected ErrAlreadyOnboarded, got %v", err)
	}
}

func TestCompleteOnboarding_OrgCreationFailure(t *testing.T) {
	svc := NewService(
		&mockOrgCreator{err: errors.New("db error")},
		&mockUserCreator{},
		&mockRoleAssigner{},
		&mockAuditLogger{},
		&mockIdentityChecker{has: false},
	)

	_, err := svc.CompleteOnboarding(context.Background(), "zit-1", "a@b.com", "Acme")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCompleteOnboarding_OwnerRoleAssignment(t *testing.T) {
	roles := &mockRoleAssigner{}
	svc := NewService(
		&mockOrgCreator{},
		&mockUserCreator{userID: "user-1"},
		roles,
		&mockAuditLogger{},
		&mockIdentityChecker{has: false},
	)

	_, err := svc.CompleteOnboarding(context.Background(), "zit-1", "a@b.com", "Acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !roles.called {
		t.Fatal("expected owner role to be assigned")
	}
}

func TestCompleteOnboarding_RoleAssignmentFailure(t *testing.T) {
	svc := NewService(
		&mockOrgCreator{},
		&mockUserCreator{userID: "user-1"},
		&mockRoleAssigner{err: errors.New("role error")},
		&mockAuditLogger{},
		&mockIdentityChecker{has: false},
	)

	_, err := svc.CompleteOnboarding(context.Background(), "zit-1", "a@b.com", "Acme")
	if err == nil {
		t.Fatal("expected error from role assignment")
	}
}
