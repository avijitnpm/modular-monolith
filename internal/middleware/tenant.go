package middleware

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func TenantContext(
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		// Prefer MembershipContext, fallback to AuthenticatedUser
		var organizationID string

		if m, ok := appcontext.GetMembership(r.Context()); ok && m.OrganizationID != "" {
			organizationID = m.OrganizationID
		} else if user, ok := appcontext.GetAuthenticatedUser(r.Context()); ok {
			organizationID = user.OrganizationID
		}

		if organizationID == "" {
			response.InternalServerError(
				w,
				"organization context missing",
			)
			return
		}

		ctx := appcontext.SetOrganizationID(
			r.Context(),
			organizationID,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
