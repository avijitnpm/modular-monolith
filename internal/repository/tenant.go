package repository

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/jackc/pgx/v5"
)

// WithTenantQuery executes fn inside a transaction where
// app.current_organization_id has been set via SET LOCAL.
func (r *Repository) WithTenantQuery(
	ctx context.Context,
	organizationID string,
	fn func(tx pgx.Tx) error,
) error {
	return database.WithTenantQuery(r.DB, ctx, organizationID, fn)
}
