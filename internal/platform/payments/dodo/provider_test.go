package dodo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

func TestVerifyWebhookSignatureValid(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("testsecret"))
	provider := &Provider{WebhookSecret: "whsec_" + secret}

	body := []byte(`{"type":"subscription.active","data":{}}`)
	msgID := "msg_123"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	sig := sign([]byte("testsecret"), msgID, ts, body)

	headers := payments.WebhookHeaders{
		ID:        msgID,
		Signature: "v1," + sig,
		Timestamp: ts,
	}

	err := provider.VerifyWebhookSignature(body, headers)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVerifyWebhookSignatureInvalid(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("testsecret"))
	provider := &Provider{WebhookSecret: "whsec_" + secret}

	body := []byte(`{"type":"subscription.active"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	headers := payments.WebhookHeaders{
		ID:        "msg_123",
		Signature: "v1,invalidsignature",
		Timestamp: ts,
	}

	err := provider.VerifyWebhookSignature(body, headers)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyWebhookSignatureExpiredTimestamp(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("testsecret"))
	provider := &Provider{WebhookSecret: "whsec_" + secret}

	body := []byte(`{"type":"test"}`)
	msgID := "msg_123"
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)

	sig := sign([]byte("testsecret"), msgID, ts, body)

	headers := payments.WebhookHeaders{
		ID:        msgID,
		Signature: "v1," + sig,
		Timestamp: ts,
	}

	err := provider.VerifyWebhookSignature(body, headers)
	if err == nil {
		t.Fatal("expected error for expired timestamp")
	}
}

func TestVerifyWebhookSignatureMissingHeaders(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("testsecret"))
	provider := &Provider{WebhookSecret: "whsec_" + secret}

	err := provider.VerifyWebhookSignature([]byte(`{}`), payments.WebhookHeaders{})
	if err == nil {
		t.Fatal("expected error for missing headers")
	}
}

func TestVerifyWebhookSignatureNoSecret(t *testing.T) {
	provider := &Provider{}

	headers := payments.WebhookHeaders{
		ID:        "msg_123",
		Signature: "v1,test",
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
	}

	err := provider.VerifyWebhookSignature([]byte(`{}`), headers)
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestVerifyWebhookSignatureMultipleSignatures(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("testsecret"))
	provider := &Provider{WebhookSecret: "whsec_" + secret}

	body := []byte(`{"data":"test"}`)
	msgID := "msg_456"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	validSig := sign([]byte("testsecret"), msgID, ts, body)

	headers := payments.WebhookHeaders{
		ID:        msgID,
		Signature: "v1,wrongsig v1," + validSig,
		Timestamp: ts,
	}

	err := provider.VerifyWebhookSignature(body, headers)
	if err != nil {
		t.Fatalf("expected nil error with multiple sigs, got %v", err)
	}
}

func sign(secret []byte, msgID, ts string, body []byte) string {
	content := fmt.Sprintf("%s.%s.%s", msgID, ts, string(body))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(content))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
