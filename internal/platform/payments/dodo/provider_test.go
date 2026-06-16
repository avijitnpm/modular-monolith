package dodo

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestVerifyWebhook_Success(t *testing.T) {
	secret := "test-secret"
	p := &Provider{WebhookSecret: secret}

	now := time.Now().UTC().Truncate(time.Second)
	payload := map[string]interface{}{
		"type": "subscription.updated",
		"data": map[string]interface{}{
			"subscription_id":    "sub_123",
			"customer_id":        "cust_456",
			"product_id":         "plan_pro",
			"status":             "active",
			"current_period_end": now.Format(time.RFC3339),
			"metadata": map[string]string{
				"organization_id": "org_789",
			},
		},
	}

	body, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	event, err := p.VerifyWebhook(context.Background(), body, sig)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.ProviderSubscriptionID != "sub_123" {
		t.Errorf("expected sub_123, got %s", event.ProviderSubscriptionID)
	}

	if event.ProviderCustomerID != "cust_456" {
		t.Errorf("expected cust_456, got %s", event.ProviderCustomerID)
	}

	if event.Plan != "plan_pro" {
		t.Errorf("expected plan_pro, got %s", event.Plan)
	}

	if event.Status != "active" {
		t.Errorf("expected active, got %s", event.Status)
	}

	if event.OrganizationID != "org_789" {
		t.Errorf("expected org_789, got %s", event.OrganizationID)
	}
}

func TestVerifyWebhook_InvalidSignature(t *testing.T) {
	p := &Provider{WebhookSecret: "test-secret"}

	body := []byte(`{"type":"subscription.updated","data":{}}`)

	_, err := p.VerifyWebhook(context.Background(), body, "bad-signature")

	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyWebhook_MissingSecret(t *testing.T) {
	p := &Provider{WebhookSecret: ""}

	_, err := p.VerifyWebhook(context.Background(), []byte(`{}`), "sig")

	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}
