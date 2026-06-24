package identity

import "time"

type Identity struct {
	ID             string
	ZitadelUserID  string
	Email          string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MembershipReference struct {
	UserID         string
	OrganizationID string
}
