package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TxFunc func(
	tx pgx.Tx,
) error

func (r *Repository) WithTransaction(
	ctx context.Context,
	fn TxFunc,
) error {

	tx, err := r.DB.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	err = fn(tx)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
