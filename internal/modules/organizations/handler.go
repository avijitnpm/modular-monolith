package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Service interface {
	RegisterOrganization(
		ctx context.Context,
		zitadelOrgID string,
		name string,
	) (*repository.Organization, error)
}

type Handler struct {
	Service Service
}

func NewHandler(
	service Service,
) *Handler {

	return &Handler{
		Service: service,
	}
}

func (h *Handler) CreateOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req CreateOrganizationRequest

	err := json.NewDecoder(r.Body).Decode(
		&req,
	)

	if err != nil {
		response.BadRequest(
			w,
			"invalid request body",
		)

		return
	}

	validationErrors := req.Validate()

	if validationErrors != nil {
		response.ValidationError(
			w,
			validationErrors,
		)

		return
	}

	org, err := h.Service.RegisterOrganization(
		r.Context(),
		req.ZitadelOrgID,
		req.Name,
	)

	if err != nil {
		if errors.Is(err, appErrors.ErrOrganizationAlreadyExists) {
			response.Error(w, http.StatusConflict, "organization already exists")
			return
		}

		response.InternalServerError(
			w,
			"failed to create organization",
		)

		return
	}

	response.Created(
		w,
		organizationResponse(
			org.ID,
			org.OrganizationID,
			org.Name,
		),
	)
}
