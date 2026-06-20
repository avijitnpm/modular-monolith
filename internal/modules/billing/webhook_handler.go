package billing

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type webhookPayload struct {
	BusinessID string      `json:"business_id"`
	Type       string      `json:"type"`
	Timestamp  string      `json:"timestamp"`
	Data       webhookData `json:"data"`
}

type webhookData struct {
	SubscriptionID   string            `json:"subscription_id"`
	CustomerID       string            `json:"customer_id"`
	ProductID        string            `json:"product_id"`
	Status           string            `json:"status"`
	CurrentPeriodEnd *time.Time        `json:"current_period_end"`
	Metadata         map[string]string `json:"metadata"`
}

func (h *Handler) HandleWebhook(
	w http.ResponseWriter,
	r *http.Request,
) {
	const maxWebhookBody = 1 << 20 // 1 MB

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		response.BadRequest(w, "failed to read request body")
		return
	}

	headers := payments.WebhookHeaders{
		ID:        r.Header.Get("webhook-id"),
		Signature: r.Header.Get("webhook-signature"),
		Timestamp: r.Header.Get("webhook-timestamp"),
	}

	err = h.Service.Provider.VerifyWebhookSignature(body, headers)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		response.BadRequest(w, "invalid webhook payload")
		return
	}

	organizationID := payload.Data.Metadata["organization_id"]
	if organizationID == "" {
		response.BadRequest(w, "missing organization_id in metadata")
		return
	}

	err = h.Service.ProcessWebhookEvent(
		r.Context(),
		organizationID,
		payload.Data.SubscriptionID,
		payload.Data.CustomerID,
		payload.Data.ProductID,
		payload.Data.Status,
		payload.Data.CurrentPeriodEnd,
	)

	if err != nil {
		response.InternalServerError(w, "failed to process webhook")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}
