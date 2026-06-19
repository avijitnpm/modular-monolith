package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

func TestServiceCreateSubscriptionTrimsFields(t *testing.T) {
	store := &fakeBillingStore{
		createSubscription: testSubscription(),
	}
	service := NewService(store, nil, nil)

	subscription, err := service.CreateSubscription(
		context.Background(),
		"org-1",
		" manual ",
		" free ",
		" active ",
	)

	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}

	if subscription.ID != "subscription-1" {
		t.Fatalf("expected subscription, got %q", subscription.ID)
	}

	if store.provider != "manual" || store.plan != "free" || store.status != "active" {
		t.Fatalf(
			"expected trimmed fields, got provider=%q plan=%q status=%q",
			store.provider,
			store.plan,
			store.status,
		)
	}
}

func TestServiceCreateCheckoutSessionTrimsPlan(t *testing.T) {
	provider := &fakeCheckoutProvider{
		url: "https://checkout.example/session-1",
	}
	service := NewService(
		&fakeBillingStore{},
		provider,
		nil,
	)

	checkoutURL, err := service.CreateCheckoutSession(
		context.Background(),
		" org-1 ",
		" prod_basic ",
	)

	if err != nil {
		t.Fatalf("CreateCheckoutSession returned error: %v", err)
	}

	if checkoutURL != provider.url {
		t.Fatalf("expected checkout url %q, got %q", provider.url, checkoutURL)
	}

	if provider.organizationID != "org-1" || provider.plan != "prod_basic" {
		t.Fatalf(
			"expected trimmed checkout args, got organization=%q plan=%q",
			provider.organizationID,
			provider.plan,
		)
	}
}

func TestServiceCreateCheckoutSessionRejectsMissingPlan(t *testing.T) {
	service := NewService(
		&fakeBillingStore{},
		&fakeCheckoutProvider{},
		nil,
	)

	_, err := service.CreateCheckoutSession(
		context.Background(),
		"org-1",
		" ",
	)

	if !errors.Is(err, ErrInvalidSubscription) {
		t.Fatalf("expected invalid subscription error, got %v", err)
	}
}

func TestServiceCreateSubscriptionRejectsMissingFields(t *testing.T) {
	service := NewService(&fakeBillingStore{}, nil, nil)

	_, err := service.CreateSubscription(
		context.Background(),
		"org-1",
		" ",
		"free",
		"active",
	)

	if !errors.Is(err, ErrInvalidSubscription) {
		t.Fatalf("expected invalid subscription error, got %v", err)
	}
}

func TestServiceUpdateSubscriptionTrimsFields(t *testing.T) {
	store := &fakeBillingStore{
		updateSubscription: testSubscription(),
	}
	service := NewService(store, nil, nil)
	currentPeriodEnd := time.Now().UTC().Add(30 * 24 * time.Hour)

	_, err := service.UpdateSubscription(
		context.Background(),
		" subscription-1 ",
		"org-1",
		" pro ",
		" active ",
		&currentPeriodEnd,
	)

	if err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}

	if store.id != "subscription-1" ||
		store.plan != "pro" ||
		store.status != "active" ||
		store.currentPeriodEnd != &currentPeriodEnd {
		t.Fatalf(
			"expected update fields, got id=%q plan=%q status=%q current_period_end=%v",
			store.id,
			store.plan,
			store.status,
			store.currentPeriodEnd,
		)
	}
}

func TestServiceUpdateSubscriptionRejectsMissingFields(t *testing.T) {
	service := NewService(&fakeBillingStore{}, nil, nil)

	_, err := service.UpdateSubscription(
		context.Background(),
		"",
		"org-1",
		"pro",
		"active",
		nil,
	)

	if !errors.Is(err, ErrInvalidSubscription) {
		t.Fatalf("expected invalid subscription error, got %v", err)
	}
}

type fakeBillingStore struct {
	getSubscription    *Subscription
	getErr             error
	createSubscription *Subscription
	createErr          error
	updateSubscription *Subscription
	updateErr          error
	id                 string
	organizationID     string
	provider           string
	plan               string
	status             string
	currentPeriodEnd   *time.Time
}

func (f *fakeBillingStore) GetSubscription(
	ctx context.Context,
	organizationID string,
) (*Subscription, error) {

	f.organizationID = organizationID

	return f.getSubscription, f.getErr
}

func (f *fakeBillingStore) CreateSubscription(
	ctx context.Context,
	organizationID string,
	provider string,
	plan string,
	status string,
) (*Subscription, error) {

	f.organizationID = organizationID
	f.provider = provider
	f.plan = plan
	f.status = status

	return f.createSubscription, f.createErr
}

func (f *fakeBillingStore) UpdateSubscription(
	ctx context.Context,
	id string,
	organizationID string,
	plan string,
	status string,
	currentPeriodEnd *time.Time,
) (*Subscription, error) {

	f.id = id
	f.organizationID = organizationID
	f.plan = plan
	f.status = status
	f.currentPeriodEnd = currentPeriodEnd

	return f.updateSubscription, f.updateErr
}

func (f *fakeBillingStore) UpsertSubscriptionByProvider(
	_ context.Context,
	_, _, _, _, _, _ string,
	_ *time.Time,
) error {
	return nil
}

func testSubscription() *Subscription {
	return &Subscription{
		ID:             "subscription-1",
		OrganizationID: "org-1",
		Provider:       "manual",
		Plan:           "free",
		Status:         "active",
	}
}

type fakeCheckoutProvider struct {
	url            string
	err            error
	organizationID string
	plan           string
}

var _ payments.Provider = (*fakeCheckoutProvider)(nil)

func (f *fakeCheckoutProvider) CreateCheckoutSession(
	ctx context.Context,
	organizationID string,
	plan string,
) (string, error) {

	f.organizationID = organizationID
	f.plan = plan

	return f.url, f.err
}

func (f *fakeCheckoutProvider) VerifyWebhookSignature(
	_ []byte,
	_ payments.WebhookHeaders,
) error {
	return nil
}

type fakeAuditLogger struct {
	events []*audit.Event
}

func (f *fakeAuditLogger) Log(_ context.Context, event *audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func TestServiceCreateSubscriptionLogsAudit(t *testing.T) {
	auditLog := &fakeAuditLogger{}
	store := &fakeBillingStore{
		createSubscription: testSubscription(),
	}
	service := NewService(store, nil, auditLog)

	_, err := service.CreateSubscription(
		context.Background(),
		"org-1",
		"manual",
		"free",
		"active",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(auditLog.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditLog.events))
	}

	e := auditLog.events[0]
	if e.Action != "subscription.created" {
		t.Errorf("expected action subscription.created, got %q", e.Action)
	}
	if e.Metadata["plan"] != "free" {
		t.Errorf("expected metadata plan=free, got %q", e.Metadata["plan"])
	}
}

func TestServiceUpdateSubscriptionLogsAudit(t *testing.T) {
	auditLog := &fakeAuditLogger{}
	store := &fakeBillingStore{
		getSubscription:    &Subscription{Status: "active", OrganizationID: "org-1"},
		updateSubscription: testSubscription(),
	}
	service := NewService(store, nil, auditLog)

	_, err := service.UpdateSubscription(
		context.Background(),
		"subscription-1",
		"org-1",
		"pro",
		"past_due",
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(auditLog.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditLog.events))
	}

	e := auditLog.events[0]
	if e.Action != "subscription.updated" {
		t.Errorf("expected action subscription.updated, got %q", e.Action)
	}
	if e.Metadata["previous_status"] != "active" {
		t.Errorf("expected previous_status=active, got %q", e.Metadata["previous_status"])
	}
	if e.Metadata["new_status"] != "past_due" {
		t.Errorf("expected new_status=past_due, got %q", e.Metadata["new_status"])
	}
}

func TestServiceCheckoutLogsAudit(t *testing.T) {
	auditLog := &fakeAuditLogger{}
	provider := &fakeCheckoutProvider{url: "https://checkout.example/s1"}
	service := NewService(&fakeBillingStore{}, provider, auditLog)

	_, err := service.CreateCheckoutSession(
		context.Background(),
		"org-1",
		"pro",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(auditLog.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditLog.events))
	}

	e := auditLog.events[0]
	if e.Action != "checkout.created" {
		t.Errorf("expected action checkout.created, got %q", e.Action)
	}
	if e.Metadata["organization_id"] != "org-1" {
		t.Errorf("expected metadata organization_id=org-1, got %q", e.Metadata["organization_id"])
	}
	if e.Metadata["plan"] != "pro" {
		t.Errorf("expected metadata plan=pro, got %q", e.Metadata["plan"])
	}
}

func TestServiceCheckoutDoesNotAuditOnFailure(t *testing.T) {
	auditLog := &fakeAuditLogger{}
	provider := &fakeCheckoutProvider{err: errors.New("provider error")}
	service := NewService(&fakeBillingStore{}, provider, auditLog)

	_, _ = service.CreateCheckoutSession(
		context.Background(),
		"org-1",
		"pro",
	)

	if len(auditLog.events) != 0 {
		t.Fatalf("expected 0 audit events on failure, got %d", len(auditLog.events))
	}
}

func TestServiceWebhookLifecycleAudit(t *testing.T) {
	cases := []struct {
		name           string
		previousStatus string
		newStatus      string
		expectedAction string
	}{
		{"activated", "trialing", "active", "subscription.activated"},
		{"past_due", "active", "past_due", "subscription.past_due"},
		{"cancelled", "active", "cancelled", "subscription.cancelled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auditLog := &fakeAuditLogger{}
			store := &fakeBillingStore{
				getSubscription: &Subscription{Status: tc.previousStatus, OrganizationID: "org-1"},
			}
			service := NewService(store, nil, auditLog)

			err := service.ProcessWebhookEvent(
				context.Background(),
				"org-1",
				"sub_123",
				"cust_456",
				"pro",
				tc.newStatus,
				nil,
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(auditLog.events) != 1 {
				t.Fatalf("expected 1 audit event, got %d", len(auditLog.events))
			}

			e := auditLog.events[0]
			if e.Action != tc.expectedAction {
				t.Errorf("expected action %q, got %q", tc.expectedAction, e.Action)
			}
			if e.Metadata["previous_status"] != tc.previousStatus {
				t.Errorf("expected previous_status=%q, got %q", tc.previousStatus, e.Metadata["previous_status"])
			}
			if e.Metadata["new_status"] != tc.newStatus {
				t.Errorf("expected new_status=%q, got %q", tc.newStatus, e.Metadata["new_status"])
			}
			if e.Metadata["provider_subscription_id"] != "sub_123" {
				t.Errorf("expected provider_subscription_id=sub_123, got %q", e.Metadata["provider_subscription_id"])
			}
		})
	}
}

func TestServiceWebhookIdempotentNoAudit(t *testing.T) {
	auditLog := &fakeAuditLogger{}
	store := &fakeBillingStore{
		getSubscription: &Subscription{Status: "active", OrganizationID: "org-1"},
	}
	service := NewService(store, nil, auditLog)

	err := service.ProcessWebhookEvent(
		context.Background(),
		"org-1",
		"sub_123",
		"cust_456",
		"pro",
		"active",
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(auditLog.events) != 0 {
		t.Fatalf("expected 0 audit events for idempotent webhook, got %d", len(auditLog.events))
	}
}

