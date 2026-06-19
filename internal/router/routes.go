package router

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/auth"
	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/middleware"
	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
	"github.com/avijitnpm/modular-monolith/internal/modules/billing"
	"github.com/avijitnpm/modular-monolith/internal/modules/organizations"
	"github.com/avijitnpm/modular-monolith/internal/modules/rbac"
	"github.com/avijitnpm/modular-monolith/internal/modules/users"
	"github.com/avijitnpm/modular-monolith/internal/platform/payments/dodo"
	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/avijitnpm/modular-monolith/internal/service"
)

func registerRoutes(
	r chi.Router,
	cfg *config.Config,
	logger *slog.Logger,
	service *service.Service,
) {

	apiTokenProvider := identity.NewZitadelProvider(
		identity.OIDCConfig{
			Issuer:   cfg.Auth.OIDCIssuer,
			Audience: cfg.Auth.OIDCAudience,
		},
	)

	oauthProvider := identity.NewZitadelProvider(
		identity.OIDCConfig{
			Issuer:      cfg.Auth.OIDCIssuer,
			Audience:    cfg.Auth.OIDCAudience,
			ClientID:    cfg.Auth.OIDCClientID,
			RedirectURL: cfg.Auth.OIDCRedirectURL,
		},
	)

	idTokenProvider := identity.NewZitadelProvider(
		identity.OIDCConfig{
			Issuer:   cfg.Auth.OIDCIssuer,
			Audience: cfg.Auth.OIDCClientID,
		},
	)

	authHandler, err := authflow.NewHandler(
		oauthProvider,
		idTokenProvider,
		service,
		cfg.Auth.SessionSecret,
		cfg.App.Env != "development",
		logger,
		cfg.App.Env == "development",
	)

	if err != nil {
		panic(err)
	}

	auditService := audit.NewService(
		service.Repository,
	)

	userHandler := users.NewHandler(
		service,
		auditService,
	)

	organizationHandler := organizations.NewHandler(
		service,
	)

	rbacRepository := rbac.NewRepository(
		service.Repository.DB,
	)

	rbacService := rbac.NewService(
		rbacRepository,
		auditService,
	)

	rbacHandler := rbac.NewHandler(
		rbacService,
	)

	billingRepository := billing.NewRepository(
		service.Repository.DB,
	)

	billingProvider := dodo.NewProvider(
		os.Getenv("DODO_API_KEY"),
		os.Getenv("DODO_WEBHOOK_SECRET"),
	)

	billingService := billing.NewService(
		billingRepository,
		billingProvider,
	)

	billingHandler := billing.NewHandler(
		billingService,
	)

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

		api.Get(
			"/auth/login",
			authHandler.Login,
		)

		api.Get(
			"/auth/callback",
			authHandler.Callback,
		)

		api.Post(
			"/auth/logout",
			authHandler.Logout,
		)

		api.Get(
			"/auth/me",
			authHandler.Me,
		)

		api.Post(
			"/organizations",
			organizationHandler.CreateOrganization,
		)

		// TEMPORARY RBAC VERIFICATION ENDPOINT
		api.Post(
			"/admin/bootstrap-rbac",
			rbacHandler.BootstrapRBAC,
		)

		// WEBHOOK (public, signature-verified)
		api.Post(
			"/billing/webhook",
			billingHandler.HandleWebhook,
		)

		// PROTECTED ROUTES

		api.Group(func(protected chi.Router) {

			protected.Use(
				auth.Middleware(apiTokenProvider),
			)

			protected.Use(
				middleware.TenantContext,
			)

			protected.Post(
				"/users",
				userHandler.RegisterUser,
			)

			protected.Get(
				"/roles",
				rbacHandler.ListRoles,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"settings.write",
				),
			).Post(
				"/roles",
				rbacHandler.CreateRole,
			)

			protected.Get(
				"/permissions",
				rbacHandler.ListPermissions,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.read",
				),
			).Get(
				"/billing",
				billingHandler.GetBilling,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.write",
				),
			).Post(
				"/billing",
				billingHandler.CreateBilling,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.write",
				),
			).Post(
				"/billing/checkout",
				billingHandler.CreateCheckout,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.write",
				),
			).Patch(
				"/billing/{id}",
				billingHandler.UpdateBilling,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"settings.write",
				),
			).Post(
				"/users/{id}/roles",
				rbacHandler.AssignRoleToUser,
			)
		})
	})
}
