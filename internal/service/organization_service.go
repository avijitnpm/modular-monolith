package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/modules/rbac"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/jackc/pgx/v5"
)

func (s *Service) RegisterOrganization(
	ctx context.Context,
	zitadelOrgID string,
	name string,
) (*repository.Organization, error) {

	var createdOrganization *repository.Organization

	err := s.WithTransaction(
		ctx,
		func(tx pgx.Tx) error {
			org, err := s.Repository.CreateOrganizationTx(
				ctx,
				tx,
				zitadelOrgID,
				name,
			)

			if err != nil {
				return err
			}

			err = rbac.BootstrapDefaultRolesTx(
				ctx,
				tx,
				org.OrganizationID,
			)

			if err != nil {
				return err
			}

			createdOrganization = org

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return createdOrganization, nil
}
