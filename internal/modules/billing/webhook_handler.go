package billing

import (
	"io"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

type WebhookHandler struct {
	Service  *Service
	Provider payments.Provider
}

func NewWebhookHandler(
	service *Service,
	provider payments.Provider,
) *WebhookHandler {

	return &WebhookHandler{
		Service:  service,
		Provider: provider,
	}
}

func (h *WebhookHandler) HandleWebhook(
	w http.ResponseWriter,
	r *http.Request,
) {

	body, err := io.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Webhook-Signature")

	event, err := h.Provider.VerifyWebhook(
		r.Context(),
		body,
		signature,
	)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := h.Service.ProcessWebhookEvent(r.Context(), event); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
