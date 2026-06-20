package billing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

type fakeWebhookProvider struct {
	verifyErr      error
	checkoutURL    string
	checkoutErr    error
	organizationID string
	plan           string
}

var _ payments.Provider = (*fakeWebhookProvider)(nil)

func (f *fakeWebhookProvider) CreateCheckoutSession(
	_ context.Context,
	organizationID string,
	plan string,
) (string, error) {
	f.organizationID = organizationID
	f.plan = plan
	return f.checkoutURL, f.checkoutErr
}

func (f *fakeWebhookProvider) VerifyWebhookSignature(
	_ []byte,
	_ payments.WebhookHeaders,
) error {
	return f.verifyErr
}

func TestHandleWebhookRejectsInvalidSignature(t *testing.T) {
	provider := &fakeWebhookProvider{
		verifyErr: fmt.Errorf("invalid signature"),
	}
	handler := NewHandler(NewService(&fakeBillingStore{}, provider, nil))

	req := webhookRequest(`{"type":"subscription.active","data":{"metadata":{"organization_id":"org-1"},"status":"active"}}`)
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestHandleWebhookRejectsInvalidJSON(t *testing.T) {
	provider := &fakeWebhookProvider{}
	handler := NewHandler(NewService(&fakeBillingStore{}, provider, nil))

	req := webhookRequest(`{invalid`)
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestHandleWebhookRejectsMissingOrgID(t *testing.T) {
	provider := &fakeWebhookProvider{}
	handler := NewHandler(NewService(&fakeBillingStore{}, provider, nil))

	req := webhookRequest(`{"type":"subscription.active","data":{"status":"active","metadata":{}}}`)
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestHandleWebhookProcessesEvent(t *testing.T) {
	store := &fakeBillingStore{
		getSubscription: &Subscription{Status: "trialing", OrganizationID: "org-1"},
	}
	provider := &fakeWebhookProvider{}
	handler := NewHandler(NewService(store, provider, nil))

	body := `{
		"business_id": "biz_123",
		"type": "subscription.active",
		"timestamp": "2026-06-19T10:00:00Z",
		"data": {
			"subscription_id": "sub_abc",
			"customer_id": "cust_xyz",
			"product_id": "prod_basic",
			"status": "active",
			"metadata": {"organization_id": "org-1"}
		}
	}`

	req := webhookRequest(body)
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	if !strings.Contains(recorder.Body.String(), `"received":true`) {
		t.Fatalf("expected received:true, got %s", recorder.Body.String())
	}
}

func TestHandleWebhookIdempotent(t *testing.T) {
	store := &fakeBillingStore{
		getSubscription: &Subscription{Status: "active", OrganizationID: "org-1"},
	}
	provider := &fakeWebhookProvider{}
	handler := NewHandler(NewService(store, provider, nil))

	body := `{
		"type": "subscription.active",
		"data": {
			"subscription_id": "sub_abc",
			"customer_id": "cust_xyz",
			"product_id": "prod_basic",
			"status": "active",
			"metadata": {"organization_id": "org-1"}
		}
	}`

	req := webhookRequest(body)
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func webhookRequest(body string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/webhook",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("webhook-id", "msg_test123")
	req.Header.Set("webhook-signature", "v1,test")
	req.Header.Set("webhook-timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	return req
}

func TestHandleWebhookRejectsOversizedBody(t *testing.T) {
	provider := &fakeWebhookProvider{}
	handler := NewHandler(NewService(&fakeBillingStore{}, provider, nil))

	// Create a body larger than 1 MB
	oversized := strings.Repeat("x", 1<<20+1)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/billing/webhook",
		strings.NewReader(oversized),
	)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
