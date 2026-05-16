package errors

var (
	ErrUserAlreadyExists = &AppError{
		Code:    "USER_ALREADY_EXISTS",
		Message: "user already exists",
	}
)
