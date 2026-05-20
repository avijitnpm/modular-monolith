package context

import "context"

const (
	OrganizationContextKey ContextKey = "organization_id"
)

func SetOrganizationID(
	ctx context.Context,
	organizationID string,
) context.Context {

	return context.WithValue(
		ctx,
		OrganizationContextKey,
		organizationID,
	)
}

func GetOrganizationID(
	ctx context.Context,
) (string, bool) {

	organizationID, ok := ctx.Value(
		OrganizationContextKey,
	).(string)

	return organizationID, ok
}
