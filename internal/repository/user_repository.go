package repository

import (
	"context"
)

func (r *Repository) CreateUser(
	ctx context.Context,
	zitadelUserID string,
	email string,
) (*User, error) {

	query := `
		INSERT INTO users (
			zitadel_user_id,
			email
		)
		VALUES ($1, $2)
		RETURNING
			id,
			zitadel_user_id,
			email,
			created_at,
			updated_at
	`

	user := &User{}

	err := r.DB.QueryRow(
		ctx,
		query,
		zitadelUserID,
		email,
	).Scan(
		&user.ID,
		&user.ZitadelUserID,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
