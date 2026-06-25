package middleware

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/authflow"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

// SessionIdentityMiddleware loads IdentityContext directly from the session cookie.
// For protected routes it enforces authentication — returns 401 if no valid session.
func SessionIdentityMiddleware(
	authHandler *authflow.Handler,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			session, err := authHandler.GetSession(r)
			if err != nil || session.User.IdentityID == "" {
				response.Error(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			ctx = appcontext.SetIdentity(ctx, &appcontext.Identity{
				IdentityID: session.User.IdentityID,
				ProviderID: session.User.Subject,
				Email:      session.User.Email,
				Name:       session.User.Name,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
