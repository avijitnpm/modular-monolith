package organizations

import (
	"context"
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type OrgGetter interface {
	GetOrganizationName(ctx context.Context, organizationID string) (string, error)
}

type SubscriptionGetter interface {
	GetSubscription(ctx context.Context, organizationID string) (plan, status string, err error)
}

type UsageGetter interface {
	GetUsage(ctx context.Context, organizationID string, metric string) (int64, error)
}

type DashboardHandler struct {
	Orgs         OrgGetter
	Subscription SubscriptionGetter
	Usage        UsageGetter
	Entitlements *entitlements.Service
}

func NewDashboardHandler(orgs OrgGetter, sub SubscriptionGetter, usage UsageGetter, ents *entitlements.Service) *DashboardHandler {
	return &DashboardHandler{Orgs: orgs, Subscription: sub, Usage: usage, Entitlements: ents}
}

func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	name, err := h.Orgs.GetOrganizationName(r.Context(), orgID)
	if err != nil {
		response.InternalServerError(w, "failed to get organization")
		return
	}

	plan, status, _ := h.Subscription.GetSubscription(r.Context(), orgID)

	var sub *DashboardSubscription
	if plan != "" {
		sub = &DashboardSubscription{Plan: plan, Status: status}
	}

	usage := h.getUsage(r.Context(), orgID)

	ents, err := h.Entitlements.GetEntitlements(r.Context(), orgID)
	if err != nil {
		response.InternalServerError(w, "failed to get entitlements")
		return
	}

	response.OK(w, DashboardResponse{
		Organization: DashboardOrgInfo{ID: orgID, Name: name},
		Subscription: sub,
		Usage:        usage,
		Entitlements: toEntitlementItems(ents),
	})
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	name, err := h.Orgs.GetOrganizationName(r.Context(), orgID)
	if err != nil {
		response.InternalServerError(w, "failed to get organization")
		return
	}

	plan, status, _ := h.Subscription.GetSubscription(r.Context(), orgID)

	response.OK(w, SummaryResponse{
		OrganizationID:   orgID,
		OrganizationName: name,
		Plan:             plan,
		Status:           status,
	})
}

func (h *DashboardHandler) UsageSummary(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appcontext.GetOrganizationID(r.Context())
	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	usage := h.getUsage(r.Context(), orgID)
	response.OK(w, usage)
}

func (h *DashboardHandler) getUsage(ctx context.Context, orgID string) DashboardUsage {
	users, _ := h.Usage.GetUsage(ctx, orgID, "users")
	docs, _ := h.Usage.GetUsage(ctx, orgID, "documents")
	api, _ := h.Usage.GetUsage(ctx, orgID, "api_requests")
	storage, _ := h.Usage.GetUsage(ctx, orgID, "storage")
	return DashboardUsage{Users: users, Documents: docs, APIRequests: api, Storage: storage}
}

func toEntitlementItems(ents []entitlements.Entitlement) []DashboardEntitlement {
	items := make([]DashboardEntitlement, 0, len(ents))
	for _, e := range ents {
		items = append(items, DashboardEntitlement{
			Metric: e.Metric, Used: e.Used, Limit: e.Limit,
			Remaining: e.Remaining, Allowed: e.Allowed,
		})
	}
	return items
}
