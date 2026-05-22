package users

import (
	"encoding/json"
	"errors"
	"net/http"

	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/service"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
	Service *service.Service
}

func NewHandler(
	service *service.Service,
) *Handler {

	return &Handler{
		Service: service,
	}
}

func (h *Handler) RegisterUser(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req RegisterUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)

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
	authenticatedUser, ok := appcontext.GetAuthenticatedUser(
		r.Context(),
	)

	if !ok {

		response.InternalServerError(
			w,
			"authenticated user missing",
		)

		return
	}

	organizationID, organizationOK := appcontext.GetOrganizationID(
		r.Context(),
	)

	if !organizationOK {

		response.InternalServerError(
			w,
			"organization context missing",
		)

		return
	}

	_ = authenticatedUser
	user, err := h.Service.RegisterUser(
		r.Context(),
		organizationID,

		req.ZitadelUserID,
		req.Email,
	)

	if err != nil {

		if errors.Is(
			err,
			appErrors.ErrUserAlreadyExists,
		) {

			response.BadRequest(
				w,
				"user already exists",
			)

			return
		}

		response.InternalServerError(
			w,
			"failed to register user",
		)

		return
	}

	res := UserResponse{
		ID:    user.ID,
		Email: user.Email,
	}

	response.Created(
		w,
		res,
	)
}
