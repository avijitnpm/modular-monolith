package middleware

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
)

// SessionIdentityMiddleware loads IdentityContext directly from the session cookie.
// This avoids a DB lookup for identity resolution when the session already contains
// the identity_id.
func SessionIdentityMiddleware(
	authHandler *authflow.Handler,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			session, err := authHandler.GetSession(r)
			if err == nil && session.User.IdentityID != "" {
				ctx = appcontext.SetIdentity(ctx, &appcontext.Identity{
					IdentityID: session.User.IdentityID,
					ProviderID: session.User.Subject,
					Email:      session.User.Email,
					Name:       session.User.Name,
				})
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
