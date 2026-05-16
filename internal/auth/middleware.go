package auth

import (
	"net/http"
	"strings"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func Middleware(
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

		claims, err := ValidateToken(tokenString)

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

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
