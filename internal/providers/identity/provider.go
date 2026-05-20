package identity

import "context"

type Provider interface {
	ValidateToken(
		ctx context.Context,
		token string,
	) (*Claims, error)
}
