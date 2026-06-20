package health

import (
	"context"
	"net/http"

	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
