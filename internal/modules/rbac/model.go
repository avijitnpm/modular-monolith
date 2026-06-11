package rbac

import "time"

type Permission struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type Role struct {
	ID             string
	OrganizationID string
	Name           string
	Permissions    []Permission
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserRole struct {
	ID             string
	OrganizationID string
	UserID         string
	RoleID         string
	CreatedAt      time.Time
}
