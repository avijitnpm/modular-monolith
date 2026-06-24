package rbac

import (
	"context"
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type PermissionChecker interface {
	UserHasPermission(
		ctx context.Context,
		organizationID string,
		membershipID string,
		permission string,
	) (bool, error)
}

func RequirePermission(
	service PermissionChecker,
	permission string,
) func(http.Handler) http.Handler {

	return func(
		next http.Handler,
	) http.Handler {

		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			organizationID, ok := appcontext.GetOrganizationID(
				r.Context(),
			)

			if !ok {
				response.InternalServerError(
					w,
					"organization context missing",
				)

				return
			}

			// Prefer MembershipContext, fallback to AuthenticatedUser.UserID
			var membershipID string
			if m, ok := appcontext.GetMembership(r.Context()); ok && m.MembershipID != "" {
				membershipID = m.MembershipID
			} else if user, ok := appcontext.GetAuthenticatedUser(r.Context()); ok {
				membershipID = user.UserID
			}

			if membershipID == "" {
				response.InternalServerError(
					w,
					"user context missing",
				)
				return
			}

			allowed, err := service.UserHasPermission(
				r.Context(),
				organizationID,
				membershipID,
				permission,
			)

			if err != nil {
				response.InternalServerError(
					w,
					"failed to check permission",
				)

				return
			}

			if !allowed {
				response.Error(
					w,
					http.StatusForbidden,
					"permission denied",
				)

				return
			}

			next.ServeHTTP(
				w,
				r,
			)
		})
	}
}
