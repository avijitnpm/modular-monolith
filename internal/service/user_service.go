package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

func (s *Service) RegisterUser(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	email string,
) (*repository.User, error) {

	return s.Repository.CreateUser(
		ctx,
		organizationID,
		zitadelUserID,
		email,
	)
}

// RegisterMembership creates a membership using an internal identity_id.
// This is the preferred path for domain services (does not require provider IDs).
func (s *Service) RegisterMembership(
	ctx context.Context,
	organizationID string,
	identityID string,
	email string,
) (*repository.User, error) {

	return s.Repository.CreateMembership(
		ctx,
		organizationID,
		identityID,
		email,
	)
}
