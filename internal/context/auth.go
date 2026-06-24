package context

// AuthenticatedUser is a legacy compatibility struct.
// New code should use IdentityContext and MembershipContext.
type AuthenticatedUser struct {
	// UserID contains the membership ID (users.id) when identity resolution
	// succeeds, or the provider ID (zitadel subject) as fallback.
	// Legacy: prefer MembershipContext.MembershipID.
	UserID string
	// OrganizationID from token claims. May be empty.
	// Legacy: prefer MembershipContext.OrganizationID.
	OrganizationID string
	// Email from token claims.
	// Legacy: prefer IdentityContext.Email.
	Email string
}
