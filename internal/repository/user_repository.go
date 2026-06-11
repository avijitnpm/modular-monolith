package repository

import (
	"context"
	"errors"

	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) FindUserByZitadelUserID(
	ctx context.Context,
	zitadelUserID string,
) (*User, error) {

	query := `
		SELECT
			id,
			zitadel_user_id,
			organization_id,
			email,
			created_at,
			updated_at
		FROM users
		WHERE zitadel_user_id = $1
	`

	user := &User{}

	err := r.DB.QueryRow(
		ctx,
		query,
		zitadelUserID,
	).Scan(
		&user.ID,
		&user.ZitadelUserID,
		&user.OrganizationID,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) CreateUser(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	email string,
) (*User, error) {

	query := `
		INSERT INTO users (
		    organization_id,
			zitadel_user_id,
			email
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			zitadel_user_id,
			organization_id,
			email,
			created_at,
			updated_at
	`

	user := &User{}

	err := r.DB.QueryRow(
		ctx,
		query,
		organizationID,
		zitadelUserID,
		email,
	).Scan(
		&user.ID,
		&user.ZitadelUserID,
		&user.OrganizationID,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {

		pgErr, ok := err.(*pgconn.PgError)

		if ok {

			if pgErr.Code == appErrors.PostgresUniqueViolation {
				return nil, appErrors.ErrUserAlreadyExists
			}
		}

		return nil, err
	}

	return user, nil
}
