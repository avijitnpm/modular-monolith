package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/avijitnpm/modular-monolith/frontend"
	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/middleware"
	"github.com/avijitnpm/modular-monolith/internal/service"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func New(
	cfg *config.Config,
	logger *slog.Logger,
	service *service.Service,
) http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)

	if cfg.OTEL.Enabled {
		r.Use(otelhttp.NewMiddleware(
			cfg.OTEL.ServiceName,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
					if pattern := routeContext.RoutePattern(); pattern != "" {
						return r.Method + " " + pattern
					}
				}

				return r.Method + " " + r.URL.Path
			}),
		))
	}

	r.Use(middleware.Logging(logger))
	r.Use(middleware.Recovery(logger))

	registerRoutes(
		r,
		cfg,
		logger,
		service,
	)

	frontendHandler := frontend.Handler()

	r.NotFound(func(
		w http.ResponseWriter,
		req *http.Request,
	) {
		if strings.HasPrefix(
			req.URL.Path,
			"/api/",
		) {
			http.NotFound(
				w,
				req,
			)
			return
		}

		frontendHandler.ServeHTTP(
			w,
			req,
		)
	})

	return r
}
