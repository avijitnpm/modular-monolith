package invitations

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
	appctx "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
	service     *Service
	authHandler *authflow.Handler
}

func NewHandler(service *Service, authHandler *authflow.Handler) *Handler {
	return &Handler{service: service, authHandler: authHandler}
}

type createRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type acceptRequest struct {
	Token string `json:"token"`
}

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := appctx.GetOrganizationID(r.Context())
	if !ok || orgID == "" {
		response.BadRequest(w, "organization context required")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Email == "" || req.Role == "" {
		response.BadRequest(w, "email and role are required")
		return
	}

	inv, err := h.service.CreateInvitation(r.Context(), orgID, req.Email, req.Role)
	if err != nil {
		if errors.Is(err, ErrInvalidRole) {
			response.BadRequest(w, "invalid role")
			return
		}
		response.InternalServerError(w, "failed to create invitation")
		return
	}

	response.Created(w, map[string]string{
		"token":      inv.Token,
		"invite_url": "/invite/" + inv.Token,
	})
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	session, err := h.authHandler.GetSession(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req acceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Token == "" {
		response.BadRequest(w, "token is required")
		return
	}

	inv, err := h.service.AcceptInvitation(r.Context(), req.Token, session.User.Subject, session.User.Email)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvitationNotFound):
			response.Error(w, http.StatusNotFound, "invitation not found")
		case errors.Is(err, ErrInvitationExpired):
			response.BadRequest(w, "invitation expired")
		case errors.Is(err, ErrEmailMismatch):
			response.Error(w, http.StatusForbidden, "email does not match invitation")
		case errors.Is(err, ErrAlreadyAccepted):
			response.Error(w, http.StatusConflict, "invitation already accepted")
		default:
			response.InternalServerError(w, "failed to accept invitation")
		}
		return
	}

	response.OK(w, map[string]string{
		"organization_id": inv.OrganizationID,
	})
}
