package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
	service     *Service
	authHandler *authflow.Handler
}

func NewHandler(service *Service, authHandler *authflow.Handler) *Handler {
	return &Handler{service: service, authHandler: authHandler}
}

type onboardingRequest struct {
	OrganizationName string `json:"organization_name"`
}

func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	session, err := h.authHandler.GetSession(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req onboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.OrganizationName == "" {
		response.BadRequest(w, "organization_name is required")
		return
	}

	result, err := h.service.CompleteOnboarding(
		r.Context(),
		session.User.Subject,
		session.User.Email,
		req.OrganizationName,
	)

	if err != nil {
		if errors.Is(err, ErrAlreadyOnboarded) {
			response.Error(w, http.StatusConflict, "identity already onboarded")
			return
		}
		response.InternalServerError(w, "onboarding failed")
		return
	}

	response.Created(w, map[string]string{
		"organization_id":   result.OrganizationID,
		"organization_name": result.OrganizationName,
	})
}
