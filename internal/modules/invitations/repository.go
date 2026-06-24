package invitations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) Create(ctx context.Context, organizationID, email, roleName string, expiresAt time.Time) (*Invitation, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	var inv Invitation
	err = database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO invitations (organization_id, email, role_name, token, expires_at)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, organization_id, email, role_name, token, expires_at, accepted_at, created_at, updated_at`,
			organizationID, email, roleName, token, expiresAt,
		).Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.RoleName, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt, &inv.UpdatedAt)
	})
	return &inv, err
}

func (r *Repository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	var inv Invitation
	err := r.DB.QueryRow(ctx,
		`SELECT id, organization_id, email, role_name, token, expires_at, accepted_at, created_at, updated_at
		 FROM invitations WHERE token = $1`,
		token,
	).Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.RoleName, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &inv, err
}

func (r *Repository) MarkAccepted(ctx context.Context, token string) error {
	_, err := r.DB.Exec(ctx,
		`UPDATE invitations SET accepted_at = now(), updated_at = now() WHERE token = $1`,
		token,
	)
	return err
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
