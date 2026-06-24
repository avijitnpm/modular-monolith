package identityresolver

import (
	"context"
	"errors"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrIdentityNotFound = errors.New("identity not found")

// Resolver resolves an identity from a provider ID (e.g. zitadel_user_id).
type Resolver interface {
	ResolveIdentity(ctx context.Context, providerID string) (*appcontext.Identity, error)
}

// IdentityResolver looks up the identities table by provider ID.
type IdentityResolver struct {
	DB *pgxpool.Pool
}

func NewIdentityResolver(db *pgxpool.Pool) *IdentityResolver {
	return &IdentityResolver{DB: db}
}

func (r *IdentityResolver) ResolveIdentity(ctx context.Context, providerID string) (*appcontext.Identity, error) {
	var id appcontext.Identity
	err := r.DB.QueryRow(ctx,
		`SELECT id, zitadel_user_id, email, name FROM identities WHERE zitadel_user_id = $1`,
		providerID,
	).Scan(&id.IdentityID, &id.ProviderID, &id.Email, &id.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrIdentityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
