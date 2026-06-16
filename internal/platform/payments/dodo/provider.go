package dodo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

const defaultBaseURL = "https://test.dodopayments.com"

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

type webhookPayload struct {
	Type string `json:"type"`
	Data struct {
		SubscriptionID string            `json:"subscription_id"`
		CustomerID     string            `json:"customer_id"`
		ProductID      string            `json:"product_id"`
		Status         string            `json:"status"`
		CurrentPeriodEnd *time.Time      `json:"current_period_end"`
		Metadata       map[string]string `json:"metadata"`
	} `json:"data"`
}

func NewProvider(
	apiKey string,
	webhookSecret string,
) *Provider {

	return &Provider{
		APIKey:        strings.TrimSpace(apiKey),
		WebhookSecret: strings.TrimSpace(webhookSecret),
		BaseURL:       defaultBaseURL,
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

func (p *Provider) VerifyWebhook(
	_ context.Context,
	body []byte,
	signature string,
) (*payments.WebhookEvent, error) {

	if p.WebhookSecret == "" {
		return nil, fmt.Errorf("webhook secret is not configured")
	}

	mac := hmac.New(sha256.New, []byte(p.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var payload webhookPayload

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}

	return &payments.WebhookEvent{
		ProviderSubscriptionID: payload.Data.SubscriptionID,
		ProviderCustomerID:     payload.Data.CustomerID,
		Plan:                   payload.Data.ProductID,
		Status:                 payload.Data.Status,
		OrganizationID:         payload.Data.Metadata["organization_id"],
		CurrentPeriodEnd:       payload.Data.CurrentPeriodEnd,
	}, nil
}
