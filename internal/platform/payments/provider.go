package payments

import "context"

type Provider interface {
	CreateCheckoutSession(
		ctx context.Context,
		organizationID string,
		plan string,
	) (string, error)
}
