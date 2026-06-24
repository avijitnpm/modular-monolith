package identityresolver

import (
	"context"
	"errors"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMembershipNotFound = errors.New("membership not found")

// MembershipResolver resolves a membership from an identity ID.
type MembershipResolver interface {
	ResolveMembership(ctx context.Context, identityID string) (*appcontext.Membership, error)
}

// MembershipService provides membership discovery for an identity.
type MembershipService interface {
	ListMemberships(ctx context.Context, identityID string) ([]appcontext.Membership, error)
	GetDefaultMembership(ctx context.Context, identityID string) (*appcontext.Membership, error)
}

// DBMembershipResolver looks up the users table by identity_id.
// If multiple memberships exist, returns the first one found.
// Org-switching is not yet supported.
type DBMembershipResolver struct {
	DB *pgxpool.Pool
}

func NewMembershipResolver(db *pgxpool.Pool) *DBMembershipResolver {
	return &DBMembershipResolver{DB: db}
}

func (r *DBMembershipResolver) ResolveMembership(ctx context.Context, identityID string) (*appcontext.Membership, error) {
	return r.GetDefaultMembership(ctx, identityID)
}

// GetDefaultMembership returns the first membership for an identity.
// No org-switching yet — returns first found.
func (r *DBMembershipResolver) GetDefaultMembership(ctx context.Context, identityID string) (*appcontext.Membership, error) {
	var m appcontext.Membership
	err := r.DB.QueryRow(ctx,
		`SELECT id, organization_id FROM users WHERE identity_id = $1 LIMIT 1`,
		identityID,
	).Scan(&m.MembershipID, &m.OrganizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMembershipNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMemberships returns all memberships for an identity.
func (r *DBMembershipResolver) ListMemberships(ctx context.Context, identityID string) ([]appcontext.Membership, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT id, organization_id FROM users WHERE identity_id = $1`,
		identityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberships []appcontext.Membership
	for rows.Next() {
		var m appcontext.Membership
		if err := rows.Scan(&m.MembershipID, &m.OrganizationID); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}
