package context

type AuthenticatedUser struct {
	UserID         string
	OrganizationID string
	Email          string
}
