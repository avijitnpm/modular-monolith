package payments

import (
	"context"
	"time"
)

type WebhookEvent struct {
	ProviderSubscriptionID string
	ProviderCustomerID     string
	Plan                   string
	Status                 string
	OrganizationID         string
	CurrentPeriodEnd       *time.Time
}

type Provider interface {
	CreateCheckoutSession(
		ctx context.Context,
		organizationID string,
		plan string,
	) (string, error)
	VerifyWebhook(
		ctx context.Context,
		body []byte,
		signature string,
	) (*WebhookEvent, error)
}
