package middleware

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/identityresolver"
)

// ResolveMembershipMiddleware resolves the default membership from the session's
// identity_id and sets MembershipContext. If no membership is found (e.g. during
// onboarding), the request proceeds without MembershipContext.
func ResolveMembershipMiddleware(
	resolver identityresolver.MembershipResolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			id, ok := appcontext.GetIdentity(ctx)
			if ok && id.IdentityID != "" {
				m, err := resolver.ResolveMembership(ctx, id.IdentityID)
				if err == nil && m != nil {
					ctx = appcontext.SetMembership(ctx, m)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
