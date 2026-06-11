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
		zitadelUserID string,
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

			authenticatedUser, ok := appcontext.GetAuthenticatedUser(
				r.Context(),
			)

			if !ok {
				response.InternalServerError(
					w,
					"authenticated user missing",
				)

				return
			}

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

			allowed, err := service.UserHasPermission(
				r.Context(),
				organizationID,
				authenticatedUser.UserID,
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
