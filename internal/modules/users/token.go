package users

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/auth"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func GenerateToken(
	w http.ResponseWriter,
	r *http.Request,
) {

	token, err := auth.GenerateToken(
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
