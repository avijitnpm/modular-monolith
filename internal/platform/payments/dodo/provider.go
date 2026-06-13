package dodo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://test.dodopayments.com"

type Provider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
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
) *Provider {

	return &Provider{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: defaultBaseURL,
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
