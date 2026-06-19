package billing

import (
	"context"
	"errors"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrInvalidSubscription       = errors.New("invalid subscription")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {

	return &Repository{
		DB: db,
	}
}

func (r *Repository) GetSubscription(
	ctx context.Context,
	organizationID string,
) (*Subscription, error) {

	var subscription Subscription
	var found bool

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		err := tx.QueryRow(
			ctx,
			`
				SELECT
					id,
					organization_id,
					provider,
					provider_customer_id,
					provider_subscription_id,
					plan,
					status,
					current_period_end,
					created_at,
					updated_at
				FROM subscriptions
				WHERE organization_id = $1
				ORDER BY created_at DESC
				LIMIT 1
			`,
			organizationID,
		).Scan(
			&subscription.ID,
			&subscription.OrganizationID,
			&subscription.Provider,
			&subscription.ProviderCustomerID,
			&subscription.ProviderSubscriptionID,
			&subscription.Plan,
			&subscription.Status,
			&subscription.CurrentPeriodEnd,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)

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

	return &subscription, nil
}

func (r *Repository) CreateSubscription(
	ctx context.Context,
	organizationID string,
	provider string,
	plan string,
	status string,
) (*Subscription, error) {

	var subscription Subscription

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				INSERT INTO subscriptions (
					organization_id,
					provider,
					plan,
					status
				)
				VALUES ($1, $2, $3, $4)
				RETURNING
					id,
					organization_id,
					provider,
					provider_customer_id,
					provider_subscription_id,
					plan,
					status,
					current_period_end,
					created_at,
					updated_at
			`,
			organizationID,
			provider,
			plan,
			status,
		).Scan(
			&subscription.ID,
			&subscription.OrganizationID,
			&subscription.Provider,
			&subscription.ProviderCustomerID,
			&subscription.ProviderSubscriptionID,
			&subscription.Plan,
			&subscription.Status,
			&subscription.CurrentPeriodEnd,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
	})

	if isUniqueViolation(err) {
		return nil, ErrSubscriptionAlreadyExists
	}

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *Repository) UpdateSubscription(
	ctx context.Context,
	id string,
	organizationID string,
	plan string,
	status string,
	currentPeriodEnd *time.Time,
) (*Subscription, error) {

	var subscription Subscription

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				UPDATE subscriptions
				SET
					plan = $3,
					status = $4,
					current_period_end = $5,
					updated_at = now()
				WHERE id = $1
					AND organization_id = $2
				RETURNING
					id,
					organization_id,
					provider,
					provider_customer_id,
					provider_subscription_id,
					plan,
					status,
					current_period_end,
					created_at,
					updated_at
			`,
			id,
			organizationID,
			plan,
			status,
			currentPeriodEnd,
		).Scan(
			&subscription.ID,
			&subscription.OrganizationID,
			&subscription.Provider,
			&subscription.ProviderCustomerID,
			&subscription.ProviderSubscriptionID,
			&subscription.Plan,
			&subscription.Status,
			&subscription.CurrentPeriodEnd,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubscriptionNotFound
	}

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *Repository) UpsertSubscriptionByProvider(
	ctx context.Context,
	organizationID string,
	provider string,
	providerSubscriptionID string,
	providerCustomerID string,
	plan string,
	status string,
	currentPeriodEnd *time.Time,
) error {

	return database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx,
			`
				INSERT INTO subscriptions (
					organization_id,
					provider,
					provider_subscription_id,
					provider_customer_id,
					plan,
					status,
					current_period_end
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (organization_id)
				DO UPDATE SET
					provider_subscription_id = EXCLUDED.provider_subscription_id,
					provider_customer_id = EXCLUDED.provider_customer_id,
					plan = EXCLUDED.plan,
					status = EXCLUDED.status,
					current_period_end = EXCLUDED.current_period_end,
					updated_at = now()
			`,
			organizationID,
			provider,
			providerSubscriptionID,
			providerCustomerID,
			plan,
			status,
			currentPeriodEnd,
		)
		return err
	})
}

func isUniqueViolation(
	err error,
) bool {

	var pgErr *pgconn.PgError

	return errors.As(
		err,
		&pgErr,
	) && pgErr.Code == "23505"
}
