package payments

import "context"

type Provider interface {
	CreateCheckoutSession(
		ctx context.Context,
		organizationID string,
		plan string,
	) (string, error)

	VerifyWebhookSignature(
		payload []byte,
		headers WebhookHeaders,
	) error
}

type WebhookHeaders struct {
	ID        string
	Signature string
	Timestamp string
}
