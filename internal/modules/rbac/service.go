package rbac

import (
	"context"
	"strings"

	"github.com/avijitnpm/modular-monolith/internal/audit"
)

type Store interface {
	ListPermissions(ctx context.Context) ([]Permission, error)
	ListRoles(ctx context.Context, organizationID string) ([]Role, error)
	CreateRole(
		ctx context.Context,
		organizationID string,
		name string,
		permissionNames []string,
	) (*Role, error)
	BootstrapDefaultRoles(
		ctx context.Context,
		organizationID string,
	) error
	AssignRoleToUser(
		ctx context.Context,
		organizationID string,
		userID string,
		roleID string,
	) (*UserRole, error)
	RemoveRoleFromUser(
		ctx context.Context,
		organizationID string,
		userID string,
		roleID string,
	) (*UserRole, error)
	UserHasPermission(
		ctx context.Context,
		organizationID string,
		membershipID string,
		permission string,
	) (bool, error)
}

type AuditLogger interface {
	Log(ctx context.Context, event *audit.Event) error
}

type Service struct {
	Repository Store
	Audit      AuditLogger
}

func NewService(
	repository Store,
	auditLogger AuditLogger,
) *Service {

	return &Service{
		Repository: repository,
		Audit:      auditLogger,
	}
}

func (s *Service) ListPermissions(
	ctx context.Context,
) ([]Permission, error) {

	return s.Repository.ListPermissions(
		ctx,
	)
}

func (s *Service) ListRoles(
	ctx context.Context,
	organizationID string,
) ([]Role, error) {

	return s.Repository.ListRoles(
		ctx,
		organizationID,
	)
}

func (s *Service) CreateRole(
	ctx context.Context,
	organizationID string,
	actorUserID string,
	name string,
	permissionNames []string,
) (*Role, error) {

	role, err := s.Repository.CreateRole(
		ctx,
		organizationID,
		strings.TrimSpace(name),
		uniqueStrings(permissionNames),
	)

	if err != nil {
		return nil, err
	}

	err = s.Audit.Log(
		ctx,
		&audit.Event{
			OrganizationID: organizationID,
			UserID:         actorUserID,
			Action:         "role_created",
			EntityType:     "role",
			EntityID:       role.ID,
		},
	)

	if err != nil {
		return nil, err
	}

	return role, nil
}

func (s *Service) BootstrapDefaultRoles(
	ctx context.Context,
	organizationID string,
) error {

	return s.Repository.BootstrapDefaultRoles(
		ctx,
		organizationID,
	)
}

func (s *Service) AssignRoleToUser(
	ctx context.Context,
	organizationID string,
	actorUserID string,
	userID string,
	roleID string,
) (*UserRole, error) {

	assignment, err := s.Repository.AssignRoleToUser(
		ctx,
		organizationID,
		userID,
		roleID,
	)

	if err != nil {
		return nil, err
	}

	err = s.Audit.Log(
		ctx,
		&audit.Event{
			OrganizationID: organizationID,
			UserID:         actorUserID,
			Action:         "role_assigned",
			EntityType:     "user_role",
			EntityID:       assignment.ID,
		},
	)

	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *Service) RemoveRoleFromUser(
	ctx context.Context,
	organizationID string,
	actorUserID string,
	userID string,
	roleID string,
) (*UserRole, error) {

	assignment, err := s.Repository.RemoveRoleFromUser(
		ctx,
		organizationID,
		userID,
		roleID,
	)

	if err != nil {
		return nil, err
	}

	err = s.Audit.Log(
		ctx,
		&audit.Event{
			OrganizationID: organizationID,
			UserID:         actorUserID,
			Action:         "role_removed",
			EntityType:     "user_role",
			EntityID:       assignment.ID,
		},
	)

	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *Service) UserHasPermission(
	ctx context.Context,
	organizationID string,
	membershipID string,
	permission string,
) (bool, error) {

	return s.Repository.UserHasPermission(
		ctx,
		organizationID,
		membershipID,
		permission,
	)
}
