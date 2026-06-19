package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetTenantContext sets app.current_organization_id for the current
// transaction using set_config with is_local=true. This is equivalent
// to SET LOCAL but supports parameterized queries.
func SetTenantContext(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
) error {
	_, err := tx.Exec(
		ctx,
		`SELECT set_config('app.current_organization_id', $1, true)`,
		organizationID,
	)
	return err
}

// WithTenantQuery executes fn inside a transaction where
// app.current_organization_id has been set via SET LOCAL.
// Any repository with access to a *pgxpool.Pool can use this.
func WithTenantQuery(
	pool *pgxpool.Pool,
	ctx context.Context,
	organizationID string,
	fn func(tx pgx.Tx) error,
) error {
	if organizationID == "" {
		return fmt.Errorf("organization ID must not be empty")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant tx: %w", err)
	}

	defer tx.Rollback(ctx)

	if err := SetTenantContext(ctx, tx, organizationID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
