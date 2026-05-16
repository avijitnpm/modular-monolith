package router

import (
	"log/slog"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/middleware"
	"github.com/avijitnpm/modular-monolith/internal/service"
	"github.com/go-chi/chi/v5"
)

func New(
	logger *slog.Logger,
	service *service.Service,
) http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(logger))
	r.Use(middleware.Recovery(logger))

	registerRoutes(r, service)

	return r
}
