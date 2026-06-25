package repository

import (
	"context"
	"errors"

	"github.com/avijitnpm/modular-monolith/internal/database"
	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreateUser(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	email string,
) (*User, error) {

	var user User

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				INSERT INTO users (
				    organization_id,
					zitadel_user_id,
					email,
					identity_id
				)
				VALUES ($1, $2, $3, (SELECT id FROM identities WHERE zitadel_user_id = $2))
				RETURNING
					id,
					identity_id,
					zitadel_user_id,
					organization_id,
					email,
					created_at,
					updated_at
			`,
			organizationID,
			zitadelUserID,
			email,
		).Scan(
			&user.ID,
			&user.IdentityID,
			&user.ZitadelUserID,
			&user.OrganizationID,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {

		pgErr, ok := err.(*pgconn.PgError)

		if ok {

			if pgErr.Code == appErrors.PostgresUniqueViolation {
				return nil, appErrors.ErrUserAlreadyExists
			}
		}

		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetByIdentityID(
	ctx context.Context,
	organizationID string,
	identityID string,
) (*User, error) {

	var user User

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`SELECT id, identity_id, zitadel_user_id, organization_id, email, created_at, updated_at
			 FROM users WHERE identity_id = $1`,
			identityID,
		).Scan(
			&user.ID,
			&user.IdentityID,
			&user.ZitadelUserID,
			&user.OrganizationID,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) ListByIdentityID(
	ctx context.Context,
	identityID string,
) ([]User, error) {

	rows, err := r.DB.Query(
		ctx,
		`SELECT id, identity_id, zitadel_user_id, organization_id, email, created_at, updated_at
		 FROM users WHERE identity_id = $1`,
		identityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.IdentityID, &u.ZitadelUserID,
			&u.OrganizationID, &u.Email, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CreateMembership creates a user/membership record using identity_id directly.
// Resolves zitadel_user_id from the identities table for backward compatibility.
func (r *Repository) CreateMembership(
	ctx context.Context,
	organizationID string,
	identityID string,
	email string,
) (*User, error) {

	var user User

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`INSERT INTO users (organization_id, identity_id, zitadel_user_id, email)
			 VALUES ($1, $2, (SELECT zitadel_user_id FROM identities WHERE id = $2), $3)
			 RETURNING id, identity_id, zitadel_user_id, organization_id, email, created_at, updated_at`,
			organizationID,
			identityID,
			email,
		).Scan(
			&user.ID,
			&user.IdentityID,
			&user.ZitadelUserID,
			&user.OrganizationID,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == appErrors.PostgresUniqueViolation {
			return nil, appErrors.ErrUserAlreadyExists
		}
		return nil, err
	}

	return &user, nil
}

// CreateMembershipWithRole atomically creates a membership and assigns a role in a single transaction.
func (r *Repository) CreateMembershipWithRole(
	ctx context.Context,
	organizationID string,
	identityID string,
	email string,
	roleName string,
) (*User, error) {

	var user User

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		err := tx.QueryRow(
			ctx,
			`INSERT INTO users (organization_id, identity_id, zitadel_user_id, email)
			 VALUES ($1, $2, (SELECT zitadel_user_id FROM identities WHERE id = $2), $3)
			 RETURNING id, identity_id, zitadel_user_id, organization_id, email, created_at, updated_at`,
			organizationID,
			identityID,
			email,
		).Scan(
			&user.ID,
			&user.IdentityID,
			&user.ZitadelUserID,
			&user.OrganizationID,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Assign role within the same transaction
		_, err = tx.Exec(ctx,
			`INSERT INTO user_roles (organization_id, user_id, role_id)
			 SELECT $1, $2, r.id FROM roles r
			 WHERE r.organization_id = $1 AND r.name = $3
			 ON CONFLICT (organization_id, user_id, role_id) DO NOTHING`,
			organizationID, user.ID, roleName,
		)
		return err
	})

	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == appErrors.PostgresUniqueViolation {
			return nil, appErrors.ErrUserAlreadyExists
		}
		return nil, err
	}

	return &user, nil
}
