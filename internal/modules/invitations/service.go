package invitations

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation expired")
	ErrEmailMismatch      = errors.New("email does not match invitation")
	ErrAlreadyAccepted    = errors.New("invitation already accepted")
	ErrInvalidRole        = errors.New("invalid role")
)

var validRoles = map[string]bool{
	"owner": true, "admin": true, "member": true, "viewer": true,
}

type Store interface {
	Create(ctx context.Context, organizationID, email, roleName string, expiresAt time.Time) (*Invitation, error)
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	MarkAccepted(ctx context.Context, token string) error
}

type UserCreator interface {
	CreateUser(ctx context.Context, organizationID, zitadelUserID, email string) (string, error)
}

type RoleAssigner interface {
	AssignRole(ctx context.Context, organizationID, userID, roleName string) error
}

type AuditLogger interface {
	Log(ctx context.Context, organizationID, action, entityType, entityID string, metadata map[string]string) error
}

type Service struct {
	Store  Store
	Users  UserCreator
	Roles  RoleAssigner
	Audit  AuditLogger
}

func NewService(store Store, users UserCreator, roles RoleAssigner, audit AuditLogger) *Service {
	return &Service{Store: store, Users: users, Roles: roles, Audit: audit}
}

func (s *Service) CreateInvitation(ctx context.Context, organizationID, email, roleName string) (*Invitation, error) {
	if !validRoles[roleName] {
		return nil, ErrInvalidRole
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := s.Store.Create(ctx, organizationID, email, roleName, expiresAt)
	if err != nil {
		return nil, err
	}
	if s.Audit != nil {
		_ = s.Audit.Log(ctx, organizationID, "invitation_created", "invitation", inv.ID, map[string]string{
			"email": email, "role": roleName,
		})
	}
	return inv, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, token, zitadelUserID, email string) (*Invitation, error) {
	inv, err := s.Store.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvitationNotFound
	}
	if inv.AcceptedAt != nil {
		return nil, ErrAlreadyAccepted
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}
	if inv.Email != email {
		return nil, ErrEmailMismatch
	}

	userID, err := s.Users.CreateUser(ctx, inv.OrganizationID, zitadelUserID, email)
	if err != nil {
		return nil, err
	}

	if err := s.Roles.AssignRole(ctx, inv.OrganizationID, userID, inv.RoleName); err != nil {
		return nil, err
	}

	if err := s.Store.MarkAccepted(ctx, token); err != nil {
		return nil, err
	}

	if s.Audit != nil {
		_ = s.Audit.Log(ctx, inv.OrganizationID, "invitation_accepted", "invitation", inv.ID, map[string]string{
			"email": email, "role": inv.RoleName, "user_id": userID,
		})
	}

	return inv, nil
}

func (s *Service) GetInvitation(ctx context.Context, token string) (*Invitation, error) {
	inv, err := s.Store.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}
