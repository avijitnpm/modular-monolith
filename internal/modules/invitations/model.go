package invitations

import "time"

type Invitation struct {
	ID             string
	OrganizationID string
	Email          string
	RoleName       string
	Token          string
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
