package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func SetTenantContext(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
) error {

	_, err := tx.Exec(
		ctx,
		`
		SET LOCAL app.current_organization_id = $1
		`,
		organizationID,
	)

	return err
}
