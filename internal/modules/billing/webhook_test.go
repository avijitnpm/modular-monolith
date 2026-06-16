package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

type mockStore struct {
	upsertCalled bool
	upsertErr    error
}

func (m *mockStore) GetSubscription(context.Context, string) (*Subscription, error) {
	return nil, nil
}

func (m *mockStore) CreateSubscription(context.Context, string, string, string, string) (*Subscription, error) {
	return nil, nil
}

func (m *mockStore) UpdateSubscription(context.Context, string, string, string, string, *time.Time) (*Subscription, error) {
	return nil, nil
}

func (m *mockStore) UpsertSubscriptionByProvider(_ context.Context, _, _, _, _, _, _ string, _ *time.Time) error {
	m.upsertCalled = true
	return m.upsertErr
}

type mockProvider struct {
	secret string
}

func (m *mockProvider) CreateCheckoutSession(context.Context, string, string) (string, error) {
	return "", nil
}

func (m *mockProvider) VerifyWebhook(_ context.Context, body []byte, signature string) (*payments.WebhookEvent, error) {
	mac := hmac.New(sha256.New, []byte(m.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, ErrInvalidSubscription
	}

	var payload struct {
		OrganizationID string `json:"organization_id"`
		Status         string `json:"status"`
	}
	json.Unmarshal(body, &payload)

	return &payments.WebhookEvent{
		OrganizationID: payload.OrganizationID,
		Status:         payload.Status,
	}, nil
}

func TestProcessWebhookEvent_Success(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store, nil)

	err := svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{
		OrganizationID:         "org_1",
		ProviderSubscriptionID: "sub_1",
		Status:                 "active",
		Plan:                   "pro",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !store.upsertCalled {
		t.Fatal("expected upsert to be called")
	}
}

func TestProcessWebhookEvent_InvalidEvent(t *testing.T) {
	svc := NewService(&mockStore{}, nil)

	err := svc.ProcessWebhookEvent(context.Background(), nil)

	if err == nil {
		t.Fatal("expected error for nil event")
	}

	err = svc.ProcessWebhookEvent(context.Background(), &payments.WebhookEvent{Status: "active"})

	if err == nil {
		t.Fatal("expected error for missing organization_id")
	}
}

func TestWebhookHandler_Returns200(t *testing.T) {
	secret := "test-secret"
	store := &mockStore{}
	svc := NewService(store, nil)
	provider := &mockProvider{secret: secret}
	handler := NewWebhookHandler(svc, provider)

	body, _ := json.Marshal(map[string]string{
		"organization_id": "org_1",
		"status":          "active",
	})

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-Webhook-Signature", sig)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWebhookHandler_Returns401OnBadSignature(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store, nil)
	provider := &mockProvider{secret: "real-secret"}
	handler := NewWebhookHandler(svc, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/webhook", strings.NewReader(`{"organization_id":"org_1","status":"active"}`))
	req.Header.Set("X-Webhook-Signature", "wrong-sig")
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
