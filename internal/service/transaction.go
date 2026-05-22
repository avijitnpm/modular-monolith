package service

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type WorkflowFunc func(
	tx pgx.Tx,
) error

func (s *Service) WithTransaction(
	ctx context.Context,
	fn WorkflowFunc,
) error {

	return s.Repository.WithTransaction(
		ctx,
		func(tx pgx.Tx) error {
			return fn(tx)
		},
	)
}
