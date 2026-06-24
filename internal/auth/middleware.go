package auth

import (
	"net/http"
	"strings"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/identityresolver"
	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func Middleware(
	provider identity.Provider,
	opts ...MiddlewareOption,
) func(http.Handler) http.Handler {

	cfg := &middlewareConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(
		next http.Handler,
	) http.Handler {

		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			header := r.Header.Get("Authorization")

			if header == "" {

				response.BadRequest(
					w,
					"missing authorization header",
				)

				return
			}

			parts := strings.Split(header, " ")

			if len(parts) != 2 {

				response.BadRequest(
					w,
					"invalid authorization header",
				)

				return
			}

			tokenString := parts[1]

			claims, err := provider.ValidateToken(
				r.Context(),
				tokenString,
			)

			if err != nil {

				response.BadRequest(
					w,
					"invalid token",
				)

				return
			}

			user := &appcontext.AuthenticatedUser{
				UserID:         claims.UserID,
				OrganizationID: claims.OrganizationID,
				Email:          claims.Email,
			}

			ctx := appcontext.SetAuthenticatedUser(
				r.Context(),
				user,
			)

			// Resolve and set IdentityContext + MembershipContext (best-effort)
			if cfg.identityResolver != nil {
				id, err := cfg.identityResolver.ResolveIdentity(ctx, claims.UserID)
				if err == nil && id != nil {
					ctx = appcontext.SetIdentity(ctx, id)

					if cfg.membershipResolver != nil {
						m, err := cfg.membershipResolver.ResolveMembership(ctx, id.IdentityID)
						if err == nil && m != nil {
							ctx = appcontext.SetMembership(ctx, m)
							// Override UserID with membershipID for domain layer compatibility
							user.UserID = m.MembershipID
							ctx = appcontext.SetAuthenticatedUser(ctx, user)
						}
					}
				}
			}

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		})
	}
}

type middlewareConfig struct {
	identityResolver   identityresolver.Resolver
	membershipResolver identityresolver.MembershipResolver
}

type MiddlewareOption func(*middlewareConfig)

func WithIdentityResolver(r identityresolver.Resolver) MiddlewareOption {
	return func(c *middlewareConfig) { c.identityResolver = r }
}

func WithMembershipResolver(r identityresolver.MembershipResolver) MiddlewareOption {
	return func(c *middlewareConfig) { c.membershipResolver = r }
}
