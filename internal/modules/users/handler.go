package users

import (
	"encoding/json"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/service"
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.Service.RegisterUser(
		r.Context(),
		req.ZitadelUserID,
		req.Email,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := UserResponse{
		ID:    user.ID,
		Email: user.Email,
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}
