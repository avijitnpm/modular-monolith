package response

import "net/http"

type SuccessResponse struct {
	Data any `json:"data"`
}

func Success(
	w http.ResponseWriter,
	status int,
	data any,
) {

	JSON(
		w,
		status,
		SuccessResponse{
			Data: data,
		},
	)
}

func OK(
	w http.ResponseWriter,
	data any,
) {

	Success(
		w,
		http.StatusOK,
		data,
	)
}

func Created(
	w http.ResponseWriter,
	data any,
) {

	Success(
		w,
		http.StatusCreated,
		data,
	)
}
