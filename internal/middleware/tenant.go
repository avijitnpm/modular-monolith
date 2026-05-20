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

		user, ok := appcontext.GetAuthenticatedUser(
			r.Context(),
		)

		if !ok {

			response.InternalServerError(
				w,
				"authenticated user missing",
			)

			return
		}

		ctx := appcontext.SetOrganizationID(
			r.Context(),
			user.OrganizationID,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
