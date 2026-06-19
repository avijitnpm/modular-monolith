package repository

import (
	"context"
	"encoding/json"
)

func (r *Repository) CreateAuditLog(
	ctx context.Context,
	organizationID string,
	userID string,
	action string,
	entityType string,
	entityID string,
	metadata map[string]string,
) error {

	var metadataJSON []byte

	if metadata != nil {
		metadataJSON, _ = json.Marshal(metadata)
	}

	query := `
		INSERT INTO audit_logs (
			organization_id,
			user_id,
			action,
			entity_type,
			entity_id,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		organizationID,
		userID,
		action,
		entityType,
		entityID,
		metadataJSON,
	)

	return err
}

type AuditLog struct {
	ID             string
	Action         string
	EntityType     string
	EntityID       *string
	UserID         *string
	Metadata       map[string]string
	CreatedAt      string
}

func (r *Repository) ListAuditLogs(
	ctx context.Context,
	organizationID string,
	limit int,
	offset int,
) ([]AuditLog, error) {

	rows, err := r.DB.Query(
		ctx,
		`
			SELECT
				id,
				action,
				entity_type,
				entity_id,
				user_id,
				metadata,
				created_at
			FROM audit_logs
			WHERE organization_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`,
		organizationID,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var logs []AuditLog

	for rows.Next() {
		var log AuditLog
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Action,
			&log.EntityType,
			&log.EntityID,
			&log.UserID,
			&metadataJSON,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			_ = json.Unmarshal(metadataJSON, &log.Metadata)
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}
