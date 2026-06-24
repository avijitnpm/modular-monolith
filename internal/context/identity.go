package context

import "context"

type Identity struct {
	IdentityID string
	ProviderID string
	Email      string
	Name       string
}

const identityContextKey ContextKey = "identity"

func SetIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey, id)
}

func GetIdentity(ctx context.Context) (*Identity, bool) {
	id, ok := ctx.Value(identityContextKey).(*Identity)
	return id, ok
}
