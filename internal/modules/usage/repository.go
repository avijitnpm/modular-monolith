package usage

import (
	"context"
	"errors"

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

func (r *Repository) GetUsage(ctx context.Context, organizationID string, metric string) (*UsageCounter, error) {
	var counter UsageCounter
	var found bool

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, organization_id, metric, value, created_at, updated_at
			 FROM usage_counters
			 WHERE organization_id = $1 AND metric = $2`,
			organizationID, metric,
		).Scan(&counter.ID, &counter.OrganizationID, &counter.Metric, &counter.Value, &counter.CreatedAt, &counter.UpdatedAt)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})

	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &counter, nil
}

func (r *Repository) IncrementUsage(ctx context.Context, organizationID string, metric string, amount int64) (*UsageCounter, error) {
	var counter UsageCounter

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO usage_counters (organization_id, metric, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (organization_id, metric)
			 DO UPDATE SET value = usage_counters.value + $3, updated_at = now()
			 RETURNING id, organization_id, metric, value, created_at, updated_at`,
			organizationID, metric, amount,
		).Scan(&counter.ID, &counter.OrganizationID, &counter.Metric, &counter.Value, &counter.CreatedAt, &counter.UpdatedAt)
	})

	if err != nil {
		return nil, err
	}
	return &counter, nil
}

func (r *Repository) SetUsage(ctx context.Context, organizationID string, metric string, value int64) (*UsageCounter, error) {
	var counter UsageCounter

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO usage_counters (organization_id, metric, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (organization_id, metric)
			 DO UPDATE SET value = $3, updated_at = now()
			 RETURNING id, organization_id, metric, value, created_at, updated_at`,
			organizationID, metric, value,
		).Scan(&counter.ID, &counter.OrganizationID, &counter.Metric, &counter.Value, &counter.CreatedAt, &counter.UpdatedAt)
	})

	if err != nil {
		return nil, err
	}
	return &counter, nil
}

func (r *Repository) ListUsage(ctx context.Context, organizationID string) ([]UsageCounter, error) {
	var counters []UsageCounter

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, organization_id, metric, value, created_at, updated_at
			 FROM usage_counters
			 WHERE organization_id = $1
			 ORDER BY metric`,
			organizationID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var c UsageCounter
			if err := rows.Scan(&c.ID, &c.OrganizationID, &c.Metric, &c.Value, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return err
			}
			counters = append(counters, c)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return counters, nil
}
