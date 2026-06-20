package errors

var (
	ErrOrganizationAlreadyExists = &AppError{
		Code:    "ORGANIZATION_ALREADY_EXISTS",
		Message: "organization already exists",
	}
)
