package billing

import (
	"context"
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type UsageStore interface {
	GetUsage(ctx context.Context, organizationID string, metric string) (int64, error)
}

type BillingAPIHandler struct {
	Billing      *Service
	Usage        UsageStore
	Entitlements *entitlements.Service
}

func NewBillingAPIHandler(billing *Service, usage UsageStore, ents *entitlements.Service) *BillingAPIHandler {
	return &BillingAPIHandler{Billing: billing, Usage: usage, Entitlements: ents}
}

func (h *BillingAPIHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	sub, err := h.Billing.GetSubscription(r.Context(), orgID)
	if err != nil {
		response.InternalServerError(w, "failed to get subscription")
		return
	}

	if sub == nil {
		response.OK(w, nil)
		return
	}

	response.OK(w, SubscriptionDetailResponse{
		Plan:             sub.Plan,
		Status:           sub.Status,
		Provider:         sub.Provider,
		CurrentPeriodEnd: sub.CurrentPeriodEnd,
	})
}

func (h *BillingAPIHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	users, err := h.Usage.GetUsage(r.Context(), orgID, "users")
	if err != nil {
		response.InternalServerError(w, "failed to get usage")
		return
	}
	docs, err := h.Usage.GetUsage(r.Context(), orgID, "documents")
	if err != nil {
		response.InternalServerError(w, "failed to get usage")
		return
	}
	api, err := h.Usage.GetUsage(r.Context(), orgID, "api_requests")
	if err != nil {
		response.InternalServerError(w, "failed to get usage")
		return
	}
	storage, err := h.Usage.GetUsage(r.Context(), orgID, "storage")
	if err != nil {
		response.InternalServerError(w, "failed to get usage")
		return
	}

	response.OK(w, UsageMetricsResponse{
		Users:       users,
		Documents:   docs,
		APIRequests: api,
		Storage:     storage,
	})
}

func (h *BillingAPIHandler) GetEntitlements(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	ents, err := h.Entitlements.GetEntitlements(r.Context(), orgID)
	if err != nil {
		response.InternalServerError(w, "failed to get entitlements")
		return
	}

	items := make([]EntitlementItem, len(ents))
	for i, e := range ents {
		items[i] = EntitlementItem{
			Metric:    e.Metric,
			Used:      e.Used,
			Limit:     e.Limit,
			Remaining: e.Remaining,
			Allowed:   e.Allowed,
		}
	}

	response.OK(w, EntitlementsResponse{Entitlements: items})
}
