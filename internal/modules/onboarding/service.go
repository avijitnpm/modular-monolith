package onboarding

import (
	"context"
	"errors"
	"fmt"

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

type UserCreatorWithRole interface {
	CreateUserWithRole(ctx context.Context, organizationID, identityID, email, roleName string) (string, error)
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
	Orgs          OrgCreator
	Users         UserCreator
	UsersWithRole UserCreatorWithRole // Atomic user+role creation (preferred)
	Roles         RoleAssigner
	Audit         AuditLogger
	Identity      IdentityChecker
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

	// Step 1: Create org + bootstrap roles (atomic)
	createdOrgID, err := s.Orgs.RegisterOrganization(ctx, orgID, organizationName)
	if err != nil {
		return nil, err
	}

	// Step 2: Create user + assign owner role (atomic when UsersWithRole is set)
	var userID string
	if s.UsersWithRole != nil {
		userID, err = s.UsersWithRole.CreateUserWithRole(ctx, createdOrgID, identityID, email, "owner")
		if err != nil {
			return nil, fmt.Errorf("onboarding: create user with role failed (org %s): %w", createdOrgID, err)
		}
	} else {
		userID, err = s.Users.CreateUser(ctx, createdOrgID, identityID, email)
		if err != nil {
			return nil, fmt.Errorf("onboarding: create user failed (org %s): %w", createdOrgID, err)
		}
		if err := s.Roles.AssignOwnerRole(ctx, createdOrgID, userID); err != nil {
			return nil, fmt.Errorf("onboarding: assign role failed (org %s, user %s): %w", createdOrgID, userID, err)
		}
	}

	// Step 3: Audit (best-effort, non-critical)
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
