package users

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func GenerateToken(
	w http.ResponseWriter,
	r *http.Request,
) {

	token, err := identity.GenerateToken(
		"user-123",
		"org-456",
		"test@example.com",
	)

	if err != nil {

		response.InternalServerError(
			w,
			"failed to generate token",
		)

		return
	}

	response.OK(
		w,
		map[string]string{
			"token": token,
		},
	)
}
