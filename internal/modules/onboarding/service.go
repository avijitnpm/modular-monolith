package onboarding

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrAlreadyOnboarded = errors.New("identity already onboarded")

type OnboardingResult struct {
	OrganizationID   string
	OrganizationName string
	UserID           string
}

type OrgCreator interface {
	RegisterOrganization(ctx context.Context, orgID string, name string) (string, error)
}

type UserCreator interface {
	CreateUser(ctx context.Context, organizationID, identityID, email string) (string, error)
}

type RoleAssigner interface {
	AssignOwnerRole(ctx context.Context, organizationID, userID string) error
}

type AuditLogger interface {
	LogOnboarding(ctx context.Context, organizationID, userID string, metadata map[string]string) error
}

type IdentityChecker interface {
	HasMembership(ctx context.Context, identityID string) (bool, error)
}

type Service struct {
	Orgs     OrgCreator
	Users    UserCreator
	Roles    RoleAssigner
	Audit    AuditLogger
	Identity IdentityChecker
}

func NewService(
	orgs OrgCreator,
	users UserCreator,
	roles RoleAssigner,
	audit AuditLogger,
	identity IdentityChecker,
) *Service {
	return &Service{
		Orgs:     orgs,
		Users:    users,
		Roles:    roles,
		Audit:    audit,
		Identity: identity,
	}
}

func (s *Service) CompleteOnboarding(ctx context.Context, identityID, email, organizationName string) (*OnboardingResult, error) {
	has, err := s.Identity.HasMembership(ctx, identityID)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, ErrAlreadyOnboarded
	}

	orgID := uuid.New().String()

	createdOrgID, err := s.Orgs.RegisterOrganization(ctx, orgID, organizationName)
	if err != nil {
		return nil, err
	}

	userID, err := s.Users.CreateUser(ctx, createdOrgID, identityID, email)
	if err != nil {
		return nil, err
	}

	if err := s.Roles.AssignOwnerRole(ctx, createdOrgID, userID); err != nil {
		return nil, err
	}

	if s.Audit != nil {
		_ = s.Audit.LogOnboarding(ctx, createdOrgID, userID, map[string]string{
			"organization_name": organizationName,
		})
	}

	return &OnboardingResult{
		OrganizationID:   createdOrgID,
		OrganizationName: organizationName,
		UserID:           userID,
	}, nil
}
