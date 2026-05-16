package response

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/pkg/validator"
)

type ErrorResponse struct {
	Error string `json:"error,omitempty"`

	ValidationErrors validator.ValidationErrors `json:"validation_errors,omitempty"`
}

func Error(
	w http.ResponseWriter,
	status int,
	message string,
) {

	JSON(
		w,
		status,
		ErrorResponse{
			Error: message,
		},
	)
}

func ValidationError(
	w http.ResponseWriter,
	errors validator.ValidationErrors,
) {

	JSON(
		w,
		http.StatusBadRequest,
		ErrorResponse{
			Error:            "validation failed",
			ValidationErrors: errors,
		},
	)
}

func BadRequest(
	w http.ResponseWriter,
	message string,
) {

	Error(
		w,
		http.StatusBadRequest,
		message,
	)
}

func InternalServerError(
	w http.ResponseWriter,
	message string,
) {

	Error(
		w,
		http.StatusInternalServerError,
		message,
	)
}
