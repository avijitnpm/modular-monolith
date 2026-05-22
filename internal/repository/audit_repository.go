package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreateAuditLog(
	ctx context.Context,
	tx pgx.Tx,
	action string,
) error {

	query := `
		INSERT INTO audit_logs (
			action
		)
		VALUES ($1)
	`

	_, err := tx.Exec(
		ctx,
		query,
		action,
	)

	return err
}
