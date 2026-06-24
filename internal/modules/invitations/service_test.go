package invitations

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockStore struct {
	inv *Invitation
	err error
}

func (m *mockStore) Create(_ context.Context, orgID, email, roleName string, expiresAt time.Time) (*Invitation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Invitation{
		ID: "inv-1", OrganizationID: orgID, Email: email,
		RoleName: roleName, Token: "tok-123", ExpiresAt: expiresAt,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockStore) GetByToken(_ context.Context, token string) (*Invitation, error) {
	return m.inv, m.err
}

func (m *mockStore) MarkAccepted(_ context.Context, token string) error { return nil }

type mockUserCreator struct {
	err error
}

func (m *mockUserCreator) CreateUser(_ context.Context, orgID, identityID, email string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "user-1", nil
}

type mockRoleAssigner struct {
	err error
}

func (m *mockRoleAssigner) AssignRole(_ context.Context, orgID, userID, roleName string) error {
	return m.err
}

type mockAudit struct{}

func (m *mockAudit) Log(_ context.Context, orgID, action, entityType, entityID string, metadata map[string]string) error {
	return nil
}

func TestCreateInvitation_Success(t *testing.T) {
	svc := NewService(&mockStore{}, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	inv, err := svc.CreateInvitation(context.Background(), "org-1", "a@b.com", "member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Token == "" {
		t.Fatal("expected token")
	}
}

func TestCreateInvitation_InvalidRole(t *testing.T) {
	svc := NewService(&mockStore{}, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	_, err := svc.CreateInvitation(context.Background(), "org-1", "a@b.com", "superadmin")
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestAcceptInvitation_Success(t *testing.T) {
	store := &mockStore{inv: &Invitation{
		ID: "inv-1", OrganizationID: "org-1", Email: "a@b.com",
		RoleName: "member", Token: "tok", ExpiresAt: time.Now().Add(time.Hour),
	}}
	svc := NewService(store, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	inv, err := svc.AcceptInvitation(context.Background(), "tok", "zit-1", "a@b.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.OrganizationID != "org-1" {
		t.Fatalf("expected org-1, got %s", inv.OrganizationID)
	}
}

func TestAcceptInvitation_Expired(t *testing.T) {
	store := &mockStore{inv: &Invitation{
		ID: "inv-1", OrganizationID: "org-1", Email: "a@b.com",
		RoleName: "member", Token: "tok", ExpiresAt: time.Now().Add(-time.Hour),
	}}
	svc := NewService(store, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	_, err := svc.AcceptInvitation(context.Background(), "tok", "zit-1", "a@b.com")
	if !errors.Is(err, ErrInvitationExpired) {
		t.Fatalf("expected ErrInvitationExpired, got %v", err)
	}
}

func TestAcceptInvitation_EmailMismatch(t *testing.T) {
	store := &mockStore{inv: &Invitation{
		ID: "inv-1", OrganizationID: "org-1", Email: "a@b.com",
		RoleName: "member", Token: "tok", ExpiresAt: time.Now().Add(time.Hour),
	}}
	svc := NewService(store, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	_, err := svc.AcceptInvitation(context.Background(), "tok", "zit-1", "other@b.com")
	if !errors.Is(err, ErrEmailMismatch) {
		t.Fatalf("expected ErrEmailMismatch, got %v", err)
	}
}

func TestAcceptInvitation_AlreadyAccepted(t *testing.T) {
	now := time.Now()
	store := &mockStore{inv: &Invitation{
		ID: "inv-1", OrganizationID: "org-1", Email: "a@b.com",
		RoleName: "member", Token: "tok", ExpiresAt: time.Now().Add(time.Hour),
		AcceptedAt: &now,
	}}
	svc := NewService(store, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	_, err := svc.AcceptInvitation(context.Background(), "tok", "zit-1", "a@b.com")
	if !errors.Is(err, ErrAlreadyAccepted) {
		t.Fatalf("expected ErrAlreadyAccepted, got %v", err)
	}
}

func TestAcceptInvitation_NotFound(t *testing.T) {
	store := &mockStore{inv: nil}
	svc := NewService(store, &mockUserCreator{}, &mockRoleAssigner{}, &mockAudit{})
	_, err := svc.AcceptInvitation(context.Background(), "tok", "zit-1", "a@b.com")
	if !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}
