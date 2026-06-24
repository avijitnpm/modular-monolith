package router

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/auth"
	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/avijitnpm/modular-monolith/internal/middleware"
	"github.com/avijitnpm/modular-monolith/internal/modules/auditmod"
	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
	"github.com/avijitnpm/modular-monolith/internal/modules/billing"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
	"github.com/avijitnpm/modular-monolith/internal/modules/health"
	"github.com/avijitnpm/modular-monolith/internal/modules/organizations"
	"github.com/avijitnpm/modular-monolith/internal/modules/rbac"
	"github.com/avijitnpm/modular-monolith/internal/modules/usage"
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

	auditHandler := auditmod.NewHandler(
		auditService,
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
		cfg.Payments.DodoAPIKey,
		cfg.Payments.WebhookSecret,
		cfg.Payments.DodoBaseURL,
	)

	billingService := billing.NewService(
		billingRepository,
		billingProvider,
		auditService,
	)

	billingHandler := billing.NewHandler(
		billingService,
	)

	usageRepository := usage.NewRepository(
		service.Repository.DB,
	)

	usageAdapter := usage.NewAdapter(usageRepository)

	entitlementsService := entitlements.NewService(
		&subscriptionAdapter{store: billingRepository},
		usageAdapter,
	)

	billingAPIHandler := billing.NewBillingAPIHandler(
		billingService,
		usageAdapter,
		entitlementsService,
	)

	dashboardHandler := organizations.NewDashboardHandler(
		&orgNameAdapter{db: service.Repository.DB},
		&subscriptionAdapter{store: billingRepository},
		usageAdapter,
		entitlementsService,
	)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	healthHandler := health.NewHandler(service.Repository.DB)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	r.Route("/api/v1", func(api chi.Router) {

		// PUBLIC ROUTES

		api.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})

		if cfg.App.Env == "development" {
			api.Get(
				"/token",
				users.GenerateToken(cfg.Auth.DevTokenSecret),
			)
		}

		api.With(middleware.PublicRateLimit()).Get(
			"/auth/login",
			authHandler.Login,
		)

		api.With(middleware.PublicRateLimit()).Get(
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

		api.With(middleware.PublicRateLimit()).Post(
			"/organizations",
			organizationHandler.CreateOrganization,
		)

		// WEBHOOK (public, signature-verified)
		api.With(middleware.WebhookRateLimit()).Post(
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
				middleware.AuthenticatedRateLimit(),
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
					"billing.read",
				),
			).Get(
				"/billing/subscription",
				billingAPIHandler.GetSubscription,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.read",
				),
			).Get(
				"/billing/usage",
				billingAPIHandler.GetUsage,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"billing.read",
				),
			).Get(
				"/billing/entitlements",
				billingAPIHandler.GetEntitlements,
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
				middleware.AuthenticatedRateLimit(),
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
				middleware.AuthenticatedRateLimit(),
			).Post(
				"/users/{id}/roles",
				rbacHandler.AssignRoleToUser,
			)

			protected.With(
				rbac.RequirePermission(
					rbacService,
					"audit.read",
				),
			).Get(
				"/audit",
				auditHandler.ListAuditLogs,
			)

			protected.Get(
				"/organizations/dashboard",
				dashboardHandler.Dashboard,
			)

			protected.Get(
				"/organizations/summary",
				dashboardHandler.Summary,
			)

			protected.Get(
				"/organizations/usage-summary",
				dashboardHandler.UsageSummary,
			)
		})
	})
}

type subscriptionAdapter struct {
	store billing.Store
}

func (a *subscriptionAdapter) GetSubscription(ctx context.Context, organizationID string) (string, string, error) {
	sub, err := a.store.GetSubscription(ctx, organizationID)
	if err != nil {
		return "", "", err
	}
	if sub == nil {
		return "", "", nil
	}
	return sub.Plan, sub.Status, nil
}

type orgNameAdapter struct {
	db *pgxpool.Pool
}

func (a *orgNameAdapter) GetOrganizationName(ctx context.Context, organizationID string) (string, error) {
	var name string
	err := database.WithTenantQuery(a.db, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT name FROM organizations WHERE organization_id = $1 LIMIT 1`,
			organizationID,
		).Scan(&name)
	})
	if err != nil {
		return "", err
	}
	return name, nil
}
