package identityresolver

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type MembershipsHandler struct {
	service MembershipService
}

func NewMembershipsHandler(service MembershipService) *MembershipsHandler {
	return &MembershipsHandler{service: service}
}

type membershipResponse struct {
	MembershipID   string `json:"membership_id"`
	OrganizationID string `json:"organization_id"`
}

func (h *MembershipsHandler) ListMemberships(w http.ResponseWriter, r *http.Request) {
	id, ok := appcontext.GetIdentity(r.Context())
	if !ok || id.IdentityID == "" {
		response.Error(w, http.StatusUnauthorized, "identity context required")
		return
	}

	memberships, err := h.service.ListMemberships(r.Context(), id.IdentityID)
	if err != nil {
		response.InternalServerError(w, "failed to list memberships")
		return
	}

	items := make([]membershipResponse, len(memberships))
	for i, m := range memberships {
		items[i] = membershipResponse{
			MembershipID:   m.MembershipID,
			OrganizationID: m.OrganizationID,
		}
	}

	response.OK(w, map[string]any{"memberships": items})
}
