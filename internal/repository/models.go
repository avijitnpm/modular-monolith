package repository

import "time"

type User struct {
	ID string

	IdentityID     string
	ZitadelUserID  string
	OrganizationID string
	Email          string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Organization struct {
	ID string

	ZitadelOrgID   string
	OrganizationID string
	Name           string

	CreatedAt time.Time
}
