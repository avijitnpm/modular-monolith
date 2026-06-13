package billing

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegrationSubscriptionLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(
		ctx,
		databaseURL,
	)

	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	defer pool.Close()

	repository := NewRepository(pool)

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	organizationID := "org-billing-" + suffix
	otherOrganizationID := "org-billing-other-" + suffix

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM subscriptions WHERE organization_id IN ($1, $2)",
			organizationID,
			otherOrganizationID,
		)
	})

	subscription, err := repository.CreateSubscription(
		ctx,
		organizationID,
		"manual",
		"free",
		"active",
	)

	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}

	if subscription.OrganizationID != organizationID {
		t.Fatalf("expected organization %q, got %q", organizationID, subscription.OrganizationID)
	}

	got, err := repository.GetSubscription(
		ctx,
		organizationID,
	)

	if err != nil {
		t.Fatalf("GetSubscription returned error: %v", err)
	}

	if got == nil || got.ID != subscription.ID {
		t.Fatalf("expected created subscription, got %#v", got)
	}

	other, err := repository.GetSubscription(
		ctx,
		otherOrganizationID,
	)

	if err != nil {
		t.Fatalf("GetSubscription for other tenant returned error: %v", err)
	}

	if other != nil {
		t.Fatalf("expected no other tenant subscription, got %#v", other)
	}

	updated, err := repository.UpdateSubscription(
		ctx,
		subscription.ID,
		organizationID,
		"pro",
		"active",
		ptrTime(time.Now().UTC().Add(30*24*time.Hour)),
	)

	if err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}

	if updated.Plan != "pro" {
		t.Fatalf("expected updated plan pro, got %q", updated.Plan)
	}

	_, err = repository.CreateSubscription(
		ctx,
		organizationID,
		"manual",
		"free",
		"active",
	)

	if !errors.Is(err, ErrSubscriptionAlreadyExists) {
		t.Fatalf("expected duplicate subscription error, got %v", err)
	}

	_, err = repository.UpdateSubscription(
		ctx,
		subscription.ID,
		otherOrganizationID,
		"pro",
		"active",
		nil,
	)

	if !errors.Is(err, ErrSubscriptionNotFound) {
		t.Fatalf("expected subscription not found, got %v", err)
	}
}

func ptrTime(
	value time.Time,
) *time.Time {

	return &value
}
