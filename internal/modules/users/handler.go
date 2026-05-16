package users

import (
	"encoding/json"
	"net/http"

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

	user, err := h.Service.RegisterUser(
		r.Context(),
		req.ZitadelUserID,
		req.Email,
	)

	if err != nil {
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
