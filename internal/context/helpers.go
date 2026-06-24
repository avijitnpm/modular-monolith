package context

import (
	"context"
)

func SetAuthenticatedUser(
	ctx context.Context,
	user *AuthenticatedUser,
) context.Context {

	return context.WithValue(
		ctx,
		UserContextKey,
		user,
	)
}

func GetAuthenticatedUser(
	ctx context.Context,
) (*AuthenticatedUser, bool) {

	user, ok := ctx.Value(
		UserContextKey,
	).(*AuthenticatedUser)

	return user, ok
}

// GetCurrentOrganizationID resolves the organization ID with priority:
// 1. MembershipContext.OrganizationID
// 2. AuthenticatedUser.OrganizationID
// 3. empty string
func GetCurrentOrganizationID(ctx context.Context) string {
	if m, ok := GetMembership(ctx); ok && m.OrganizationID != "" {
		return m.OrganizationID
	}
	if user, ok := GetAuthenticatedUser(ctx); ok && user.OrganizationID != "" {
		return user.OrganizationID
	}
	return ""
}
