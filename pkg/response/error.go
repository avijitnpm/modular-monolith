package response

import (
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
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
