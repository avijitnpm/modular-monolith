package service

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

func (s *Service) RegisterUser(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	email string,
) (*repository.User, error) {

	var createdUser *repository.User

	err := s.WithTransaction(
		ctx,
		func(tx pgx.Tx) error {

			user, err := s.Repository.CreateUser(
				ctx,
				organizationID,
				zitadelUserID,
				email,
			)

			if err != nil {
				return err
			}

			err = s.Repository.CreateAuditLog(
				ctx,
				tx,
				"user_registered",
			)

			if err != nil {
				return err
			}

			createdUser = user

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return createdUser, nil
}
