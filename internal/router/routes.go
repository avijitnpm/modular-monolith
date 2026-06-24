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
	identitymod "github.com/avijitnpm/modular-monolith/internal/modules/identity"
	"github.com/avijitnpm/modular-monolith/internal/modules/invitations"
	"github.com/avijitnpm/modular-monolith/internal/modules/onboarding"
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

	identityRepository := identitymod.NewRepository(service.Repository.DB)
	identityService := identitymod.NewService(identityRepository)

	authHandler, err := authflow.NewHandler(
		oauthProvider,
		idTokenProvider,
		&identityServiceAdapter{svc: identityService},
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

	onboardingService := onboarding.NewService(
		&onboardingOrgAdapter{svc: service},
		&onboardingUserAdapter{svc: service},
		&onboardingRoleAdapter{rbacRepo: rbacRepository},
		&onboardingAuditAdapter{audit: auditService},
		&onboardingIdentityChecker{db: service.Repository.DB},
	)
	onboardingHandler := onboarding.NewHandler(onboardingService, authHandler)

	invitationsRepo := invitations.NewRepository(service.Repository.DB)
	invitationsService := invitations.NewService(
		invitationsRepo,
		&onboardingUserAdapter{svc: service},
		&invitationsRoleAdapter{rbacRepo: rbacRepository},
		&invitationsAuditAdapter{audit: auditService},
	)
	invitationsHandler := invitations.NewHandler(invitationsService, authHandler)

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

		api.With(middleware.PublicRateLimit()).Post(
			"/onboarding",
			onboardingHandler.CompleteOnboarding,
		)

		api.With(middleware.PublicRateLimit()).Post(
			"/invitations/accept",
			invitationsHandler.AcceptInvitation,
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

			protected.Post(
				"/invitations",
				invitationsHandler.CreateInvitation,
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

type identityServiceAdapter struct {
	svc *identitymod.Service
}

func (a *identityServiceAdapter) FindOrCreateIdentity(ctx context.Context, zitadelUserID, email, name string) error {
	_, err := a.svc.FindOrCreateIdentity(ctx, zitadelUserID, email, name)
	return err
}

type onboardingOrgAdapter struct {
	svc *service.Service
}

func (a *onboardingOrgAdapter) RegisterOrganization(ctx context.Context, orgID, name string) (string, error) {
	org, err := a.svc.RegisterOrganization(ctx, orgID, name)
	if err != nil {
		return "", err
	}
	return org.OrganizationID, nil
}

type onboardingUserAdapter struct {
	svc *service.Service
}

func (a *onboardingUserAdapter) CreateUser(ctx context.Context, organizationID, zitadelUserID, email string) (string, error) {
	user, err := a.svc.RegisterUser(ctx, organizationID, zitadelUserID, email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

type onboardingRoleAdapter struct {
	rbacRepo *rbac.Repository
}

func (a *onboardingRoleAdapter) AssignOwnerRole(ctx context.Context, organizationID, userID string) error {
	roles, err := a.rbacRepo.ListRoles(ctx, organizationID)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.Name == "owner" {
			_, err = a.rbacRepo.AssignRoleToUser(ctx, organizationID, userID, role.ID)
			return err
		}
	}
	return nil
}

type onboardingAuditAdapter struct {
	audit *audit.Service
}

func (a *onboardingAuditAdapter) LogOnboarding(ctx context.Context, organizationID, userID string, metadata map[string]string) error {
	return a.audit.Log(ctx, &audit.Event{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         "onboarding_completed",
		EntityType:     "organization",
		EntityID:       organizationID,
		Metadata:       metadata,
	})
}

type onboardingIdentityChecker struct {
	db *pgxpool.Pool
}

func (a *onboardingIdentityChecker) HasOrganization(ctx context.Context, zitadelUserID string) (bool, error) {
	var count int
	err := a.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE zitadel_user_id = $1`,
		zitadelUserID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

type invitationsRoleAdapter struct {
	rbacRepo *rbac.Repository
}

func (a *invitationsRoleAdapter) AssignRole(ctx context.Context, organizationID, userID, roleName string) error {
	roles, err := a.rbacRepo.ListRoles(ctx, organizationID)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.Name == roleName {
			_, err = a.rbacRepo.AssignRoleToUser(ctx, organizationID, userID, role.ID)
			return err
		}
	}
	return nil
}

type invitationsAuditAdapter struct {
	audit *audit.Service
}

func (a *invitationsAuditAdapter) Log(ctx context.Context, organizationID, action, entityType, entityID string, metadata map[string]string) error {
	return a.audit.Log(ctx, &audit.Event{
		OrganizationID: organizationID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		Metadata:       metadata,
	})
}
