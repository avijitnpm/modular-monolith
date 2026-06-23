package dodo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

type Provider struct {
	APIKey        string
	WebhookSecret string
	BaseURL       string
	HTTPClient    *http.Client
}

type checkoutRequest struct {
	ProductCart []productItem     `json:"product_cart"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type productItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type checkoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

func NewProvider(
	apiKey string,
	webhookSecret string,
	baseURL string,
) *Provider {

	return &Provider{
		APIKey:        strings.TrimSpace(apiKey),
		WebhookSecret: strings.TrimSpace(webhookSecret),
		BaseURL:       strings.TrimSpace(baseURL),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *Provider) CreateCheckoutSession(
	ctx context.Context,
	organizationID string,
	plan string,
) (string, error) {

	if p.APIKey == "" {
		return "", fmt.Errorf("dodo api key is required")
	}

	payload, err := json.Marshal(
		checkoutRequest{
			ProductCart: []productItem{
				{
					ProductID: plan,
					Quantity:  1,
				},
			},
			Metadata: map[string]string{
				"organization_id": organizationID,
			},
		},
	)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(p.BaseURL, "/")+"/checkouts",
		bytes.NewReader(payload),
	)

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.HTTPClient.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("dodo checkout failed with status %d", resp.StatusCode)
	}

	var checkout checkoutResponse

	if err := json.NewDecoder(resp.Body).Decode(&checkout); err != nil {
		return "", err
	}

	if strings.TrimSpace(checkout.CheckoutURL) == "" {
		return "", fmt.Errorf("dodo checkout response missing checkout_url")
	}

	return checkout.CheckoutURL, nil
}

const webhookTimestampTolerance = 5 * time.Minute

func (p *Provider) VerifyWebhookSignature(
	payload []byte,
	headers payments.WebhookHeaders,
) error {

	if p.WebhookSecret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}

	if headers.ID == "" || headers.Signature == "" || headers.Timestamp == "" {
		return fmt.Errorf("missing required webhook headers")
	}

	ts, err := strconv.ParseInt(headers.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook timestamp")
	}

	now := time.Now().Unix()
	if math.Abs(float64(now-ts)) > webhookTimestampTolerance.Seconds() {
		return fmt.Errorf("webhook timestamp too old or too new")
	}

	secret := p.WebhookSecret
	if strings.HasPrefix(secret, "whsec_") {
		secret = secret[6:]
	}

	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return fmt.Errorf("invalid webhook secret encoding")
	}

	signedContent := fmt.Sprintf("%s.%s.%s", headers.ID, headers.Timestamp, string(payload))

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, sig := range strings.Split(headers.Signature, " ") {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == "v1" && hmac.Equal([]byte(parts[1]), []byte(expected)) {
			return nil
		}
	}

	return fmt.Errorf("webhook signature verification failed")
}
