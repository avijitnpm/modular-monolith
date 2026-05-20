package repository

import (
	"context"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/database"
)

func (r *Repository) WithTenantContext(
	ctx context.Context,
) error {

	organizationID, ok := appcontext.GetOrganizationID(
		ctx,
	)

	if !ok {
		return nil
	}

	tx, err := r.DB.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	err = database.SetTenantContext(
		ctx,
		tx,
		organizationID,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
