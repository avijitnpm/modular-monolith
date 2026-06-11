package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/audit"
)

func TestServiceCreateRoleLogsAudit(t *testing.T) {
	store := &fakeStore{
		createRole: &Role{
			ID:             "role-1",
			OrganizationID: "org-1",
			Name:           "support",
		},
	}
	auditLogger := &fakeAuditLogger{}
	service := NewService(
		store,
		auditLogger,
	)

	role, err := service.CreateRole(
		context.Background(),
		"org-1",
		"actor-1",
		" support ",
		[]string{"users.read", "users.read"},
	)

	if err != nil {
		t.Fatalf("CreateRole returned error: %v", err)
	}

	if role.ID != "role-1" {
		t.Fatalf("expected created role, got %q", role.ID)
	}

	if store.createRoleName != "support" {
		t.Fatalf("expected trimmed role name, got %q", store.createRoleName)
	}

	if len(store.createRolePermissions) != 1 || store.createRolePermissions[0] != "users.read" {
		t.Fatalf("expected unique permissions, got %#v", store.createRolePermissions)
	}

	assertAuditEvent(
		t,
		auditLogger.events,
		"role_created",
		"role",
		"role-1",
	)
}

func TestServiceCreateRoleReturnsRepositoryErrorWithoutAudit(t *testing.T) {
	store := &fakeStore{
		createRoleErr: ErrRoleAlreadyExists,
	}
	auditLogger := &fakeAuditLogger{}
	service := NewService(
		store,
		auditLogger,
	)

	_, err := service.CreateRole(
		context.Background(),
		"org-1",
		"actor-1",
		"support",
		nil,
	)

	if !errors.Is(err, ErrRoleAlreadyExists) {
		t.Fatalf("expected role duplicate error, got %v", err)
	}

	if len(auditLogger.events) != 0 {
		t.Fatalf("expected no audit event, got %#v", auditLogger.events)
	}
}

func TestServiceCreateRoleReturnsUnknownPermissions(t *testing.T) {
	store := &fakeStore{
		createRoleErr: ErrUnknownPermissions,
	}
	service := NewService(
		store,
		&fakeAuditLogger{},
	)

	_, err := service.CreateRole(
		context.Background(),
		"org-1",
		"actor-1",
		"support",
		[]string{"missing.permission"},
	)

	if !errors.Is(err, ErrUnknownPermissions) {
		t.Fatalf("expected unknown permissions, got %v", err)
	}
}

func TestServiceAssignRoleToUserLogsAudit(t *testing.T) {
	store := &fakeStore{
		assignRole: &UserRole{
			ID:             "assignment-1",
			OrganizationID: "org-1",
			UserID:         "user-1",
			RoleID:         "role-1",
		},
	}
	auditLogger := &fakeAuditLogger{}
	service := NewService(
		store,
		auditLogger,
	)

	assignment, err := service.AssignRoleToUser(
		context.Background(),
		"org-1",
		"actor-1",
		"user-1",
		"role-1",
	)

	if err != nil {
		t.Fatalf("AssignRoleToUser returned error: %v", err)
	}

	if assignment.ID != "assignment-1" {
		t.Fatalf("expected assignment, got %q", assignment.ID)
	}

	assertAuditEvent(
		t,
		auditLogger.events,
		"role_assigned",
		"user_role",
		"assignment-1",
	)
}

func TestServiceRemoveRoleFromUserLogsAudit(t *testing.T) {
	store := &fakeStore{
		removeRole: &UserRole{
			ID:             "assignment-1",
			OrganizationID: "org-1",
			UserID:         "user-1",
			RoleID:         "role-1",
		},
	}
	auditLogger := &fakeAuditLogger{}
	service := NewService(
		store,
		auditLogger,
	)

	assignment, err := service.RemoveRoleFromUser(
		context.Background(),
		"org-1",
		"actor-1",
		"user-1",
		"role-1",
	)

	if err != nil {
		t.Fatalf("RemoveRoleFromUser returned error: %v", err)
	}

	if assignment.ID != "assignment-1" {
		t.Fatalf("expected assignment, got %q", assignment.ID)
	}

	assertAuditEvent(
		t,
		auditLogger.events,
		"role_removed",
		"user_role",
		"assignment-1",
	)
}

func TestServiceAssignRoleToUserReturnsUnknownRole(t *testing.T) {
	store := &fakeStore{
		assignRoleErr: ErrRoleNotFound,
	}
	service := NewService(
		store,
		&fakeAuditLogger{},
	)

	_, err := service.AssignRoleToUser(
		context.Background(),
		"org-1",
		"actor-1",
		"user-1",
		"missing-role",
	)

	if !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expected role not found, got %v", err)
	}
}

func assertAuditEvent(
	t *testing.T,
	events []*audit.Event,
	action string,
	entityType string,
	entityID string,
) {

	t.Helper()

	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %#v", events)
	}

	event := events[0]

	if event.Action != action {
		t.Fatalf("expected action %q, got %q", action, event.Action)
	}

	if event.EntityType != entityType {
		t.Fatalf("expected entity type %q, got %q", entityType, event.EntityType)
	}

	if event.EntityID != entityID {
		t.Fatalf("expected entity id %q, got %q", entityID, event.EntityID)
	}
}

type fakeStore struct {
	createRole            *Role
	createRoleErr         error
	createRoleName        string
	createRolePermissions []string

	assignRole    *UserRole
	assignRoleErr error

	removeRole    *UserRole
	removeRoleErr error
}

func (f *fakeStore) ListPermissions(
	context.Context,
) ([]Permission, error) {

	return nil, nil
}

func (f *fakeStore) ListRoles(
	context.Context,
	string,
) ([]Role, error) {

	return nil, nil
}

func (f *fakeStore) CreateRole(
	_ context.Context,
	_ string,
	name string,
	permissionNames []string,
) (*Role, error) {

	f.createRoleName = name
	f.createRolePermissions = permissionNames

	if f.createRoleErr != nil {
		return nil, f.createRoleErr
	}

	return f.createRole, nil
}

func (f *fakeStore) BootstrapDefaultRoles(
	context.Context,
	string,
) error {

	return nil
}

func (f *fakeStore) AssignRoleToUser(
	context.Context,
	string,
	string,
	string,
) (*UserRole, error) {

	if f.assignRoleErr != nil {
		return nil, f.assignRoleErr
	}

	return f.assignRole, nil
}

func (f *fakeStore) RemoveRoleFromUser(
	context.Context,
	string,
	string,
	string,
) (*UserRole, error) {

	if f.removeRoleErr != nil {
		return nil, f.removeRoleErr
	}

	return f.removeRole, nil
}

func (f *fakeStore) UserHasPermission(
	context.Context,
	string,
	string,
	string,
) (bool, error) {

	return false, nil
}

type fakeAuditLogger struct {
	events []*audit.Event
	err    error
}

func (f *fakeAuditLogger) Log(
	_ context.Context,
	event *audit.Event,
) error {

	if f.err != nil {
		return f.err
	}

	f.events = append(
		f.events,
		event,
	)

	return nil
}
