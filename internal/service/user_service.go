package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

func (s *Service) RegisterUser(
	ctx context.Context,
	zitadelUserID string,
	email string,
) (*repository.User, error) {

	user, err := s.Repository.CreateUser(
		ctx,
		zitadelUserID,
		email,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
