package repository

import (
	"context"
)

func (r *Repository) CreateOrganization(
	ctx context.Context,
	zitadelOrgID string,
	name string,
) (*Organization, error) {

	query := `
		INSERT INTO organizations (
			zitadel_org_id,
			name
		)
		VALUES ($1, $2)
		RETURNING
			id,
			zitadel_org_id,
			name,
			created_at
	`

	org := &Organization{}

	err := r.DB.QueryRow(
		ctx,
		query,
		zitadelOrgID,
		name,
	).Scan(
		&org.ID,
		&org.ZitadelOrgID,
		&org.Name,
		&org.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return org, nil
}
