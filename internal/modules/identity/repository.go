package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetByZitadelUserID(ctx context.Context, zitadelUserID string) (*Identity, error) {
	var id Identity
	err := r.DB.QueryRow(ctx,
		`SELECT id, zitadel_user_id, email, name, created_at, updated_at
		 FROM identities WHERE zitadel_user_id = $1`, zitadelUserID,
	).Scan(&id.ID, &id.ZitadelUserID, &id.Email, &id.Name, &id.CreatedAt, &id.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &id, err
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*Identity, error) {
	var id Identity
	err := r.DB.QueryRow(ctx,
		`SELECT id, zitadel_user_id, email, name, created_at, updated_at
		 FROM identities WHERE email = $1`, email,
	).Scan(&id.ID, &id.ZitadelUserID, &id.Email, &id.Name, &id.CreatedAt, &id.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &id, err
}

func (r *Repository) Create(ctx context.Context, zitadelUserID, email, name string) (*Identity, error) {
	var id Identity
	err := r.DB.QueryRow(ctx,
		`INSERT INTO identities (zitadel_user_id, email, name)
		 VALUES ($1, $2, $3)
		 RETURNING id, zitadel_user_id, email, name, created_at, updated_at`,
		zitadelUserID, email, name,
	).Scan(&id.ID, &id.ZitadelUserID, &id.Email, &id.Name, &id.CreatedAt, &id.UpdatedAt)
	return &id, err
}

func (r *Repository) Update(ctx context.Context, zitadelUserID, email, name string) (*Identity, error) {
	var id Identity
	err := r.DB.QueryRow(ctx,
		`UPDATE identities SET email = $2, name = $3, updated_at = now()
		 WHERE zitadel_user_id = $1
		 RETURNING id, zitadel_user_id, email, name, created_at, updated_at`,
		zitadelUserID, email, name,
	).Scan(&id.ID, &id.ZitadelUserID, &id.Email, &id.Name, &id.CreatedAt, &id.UpdatedAt)
	return &id, err
}

func (r *Repository) GetMemberships(ctx context.Context, identityID string) ([]MembershipReference, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT id, organization_id FROM users WHERE identity_id = $1`, identityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberships []MembershipReference
	for rows.Next() {
		var m MembershipReference
		if err := rows.Scan(&m.UserID, &m.OrganizationID); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}
