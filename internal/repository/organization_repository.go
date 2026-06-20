package repository

import (
	"context"

	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

type organizationQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) CreateOrganization(
	ctx context.Context,
	zitadelOrgID string,
	name string,
) (*Organization, error) {

	return r.CreateOrganizationTx(
		ctx,
		r.DB,
		zitadelOrgID,
		name,
	)
}

func (r *Repository) CreateOrganizationTx(
	ctx context.Context,
	q organizationQueryer,
	zitadelOrgID string,
	name string,
) (*Organization, error) {

	query := `
		INSERT INTO organizations (
			zitadel_org_id,
			organization_id,
			name
		)
		VALUES ($1, $1, $2)
		RETURNING
			id,
			zitadel_org_id,
			organization_id,
			name,
			created_at
	`

	org := &Organization{}

	err := q.QueryRow(
		ctx,
		query,
		zitadelOrgID,
		name,
	).Scan(
		&org.ID,
		&org.ZitadelOrgID,
		&org.OrganizationID,
		&org.Name,
		&org.CreatedAt,
	)

	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == appErrors.PostgresUniqueViolation {
			return nil, appErrors.ErrOrganizationAlreadyExists
		}
		return nil, err
	}

	return org, nil
}
