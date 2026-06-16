package billing

import (
	"context"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

func TestValidateTransition_Allowed(t *testing.T) {
	cases := [][2]string{
		{"trialing", "active"},
		{"trialing", "cancelled"},
		{"active", "past_due"},
		{"active", "cancelled"},
		{"past_due", "active"},
		{"past_due", "cancelled"},
		{"cancelled", "expired"},
	}

	for _, tc := range cases {
		if err := ValidateTransition(tc[0], tc[1]); err != nil {
			t.Errorf("expected %s -> %s to be allowed, got %v", tc[0], tc[1], err)
		}
	}
}

func TestValidateTransition_Forbidden(t *testing.T) {
	cases := [][2]string{
		{"expired", "active"},
		{"expired", "trialing"},
		{"cancelled", "active"},
		{"cancelled", "trialing"},
		{"active", "trialing"},
		{"past_due", "trialing"},
	}

	for _, tc := range cases {
		if err := ValidateTransition(tc[0], tc[1]); err == nil {
			t.Errorf("expected %s -> %s to be forbidden", tc[0], tc[1])
		}
	}
}

func TestValidateTransition_SameStatus(t *testing.T) {
	statuses := []string{"trialing", "active", "past_due", "cancelled", "expired"}

	for _, s := range statuses {
		if err := ValidateTransition(s, s); err != nil {
			t.Errorf("expected %s -> %s (same) to be allowed, got %v", s, s, err)
		}
	}
}

type lifecycleStore struct {
	subscription *Subscription
	upsertCalled bool
}

func (s *lifecycleStore) GetSubscription(_ context.Context, _ string) (*Subscription, error) {
	return s.subscription, nil
}

func (s *lifecycleStore) CreateSubscription(_ context.Context, _, _, _, _ string) (*Subscription, error) {
	return nil, nil
}

func (s *lifecycleStore) UpdateSubscription(_ context.Context, _, _, _, _ string, _ *time.Time) (*Subscription, error) {
	return nil, nil
}

func (s *lifecycleStore) UpsertSubscriptionByProvider(_ context.Context, _, _, _, _, _, _ string, _ *time.Time) error {
	s.upsertCalled = true
	return nil
}

func TestProcessWebhookEvent_Idempotent(t *testing.T) {
	store := &lifecycleStore{
		subscription: &Subscription{Status: "active"},
	}
	svc := NewService(store, nil)

	err := svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{
		OrganizationID: "org_1",
		Status:         "active",
	})

	if err != nil {
		t.Fatalf("expected no error on duplicate, got %v", err)
	}

	if store.upsertCalled {
		t.Fatal("expected upsert NOT to be called for idempotent event")
	}
}

func TestProcessWebhookEvent_InvalidTransition(t *testing.T) {
	store := &lifecycleStore{
		subscription: &Subscription{Status: "expired"},
	}
	svc := NewService(store, nil)

	err := svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{
		OrganizationID: "org_1",
		Status:         "active",
	})

	if err != ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	if store.upsertCalled {
		t.Fatal("expected upsert NOT to be called for invalid transition")
	}
}

func TestProcessWebhookEvent_ValidTransition(t *testing.T) {
	store := &lifecycleStore{
		subscription: &Subscription{Status: "active"},
	}
	svc := NewService(store, nil)

	err := svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{
		OrganizationID: "org_1",
		Status:         "past_due",
		Plan:           "pro",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !store.upsertCalled {
		t.Fatal("expected upsert to be called")
	}
}

func TestProcessWebhookEvent_NewSubscription(t *testing.T) {
	store := &lifecycleStore{subscription: nil}
	svc := NewService(store, nil)

	err := svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{
		OrganizationID: "org_new",
		Status:         "trialing",
		Plan:           "starter",
	})

	if err != nil {
		t.Fatalf("expected no error for new subscription, got %v", err)
	}

	if !store.upsertCalled {
		t.Fatal("expected upsert to be called for new subscription")
	}
}
