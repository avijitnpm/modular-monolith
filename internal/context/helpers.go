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
