package repository

import (
	"context"
)

func (r *Repository) CreateAuditLog(
	ctx context.Context,
	organizationID string,
	userID string,
	action string,
	entityType string,
	entityID string,
) error {

	query := `
		INSERT INTO audit_logs (
			organization_id,
			user_id,
			action,
			entity_type,
			entity_id
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		organizationID,
		userID,
		action,
		entityType,
		entityID,
	)

	return err
}
