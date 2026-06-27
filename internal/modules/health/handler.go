package health

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db     Pinger
	logger *slog.Logger
}

func NewHandler(db Pinger, opts ...Option) *Handler {
	h := &Handler{db: db, logger: slog.Default()}
	for _, o := range opts {
		o(h)
	}
	return h
}

type Option func(*Handler)

func WithLogger(l *slog.Logger) Option {
	return func(h *Handler) { h.logger = l }
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"readiness check failed",
			"dependency", "postgres",
			"error", err.Error(),
		)
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
