package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/avijitnpm/modular-monolith/internal/auth"
	"github.com/avijitnpm/modular-monolith/internal/modules/users"
	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/avijitnpm/modular-monolith/internal/service"
)

func registerRoutes(
	r chi.Router,
	service *service.Service,
) {

	userHandler := users.NewHandler(service)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(api chi.Router) {

		// PUBLIC ROUTES

		api.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})

		api.Get(
			"/token",
			users.GenerateToken,
		)

		// PROTECTED ROUTES

		api.Group(func(protected chi.Router) {

			provider := identity.NewZitadelProvider()

			protected.Use(
				auth.Middleware(provider),
			)

			protected.Post(
				"/users",
				userHandler.RegisterUser,
			)
		})
	})
}
