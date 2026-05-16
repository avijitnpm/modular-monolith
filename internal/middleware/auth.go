package middleware

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

func MockAuth(
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		user := &appcontext.AuthenticatedUser{
			UserID:         "user-123",
			OrganizationID: "org-456",
			Email:          "test@example.com",
		}

		ctx := appcontext.SetAuthenticatedUser(
			r.Context(),
			user,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
