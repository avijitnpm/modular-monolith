package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

func (s *Service) RegisterOrganization(
	ctx context.Context,
	zitadelOrgID string,
	name string,
) (*repository.Organization, error) {

	org, err := s.Repository.CreateOrganization(
		ctx,
		zitadelOrgID,
		name,
	)

	if err != nil {
		return nil, err
	}

	return org, nil
}
