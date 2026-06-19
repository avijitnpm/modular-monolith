package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/audit"
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

	if s.Audit != nil {
		_ = s.Audit.Log(ctx, &audit.Event{
			OrganizationID: createdOrganization.OrganizationID,
			Action:         "organization_created",
			EntityType:     "organization",
			EntityID:       createdOrganization.OrganizationID,
			Metadata: map[string]string{
				"name":           createdOrganization.Name,
				"zitadel_org_id": createdOrganization.ZitadelOrgID,
			},
		})
	}

	return createdOrganization, nil
}
