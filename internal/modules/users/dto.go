package users

import "github.com/avijitnpm/modular-monolith/pkg/validator"

type RegisterUserRequest struct {
	ZitadelUserID string `json:"zitadel_user_id"`
	Email         string `json:"email"`
}

func (r *RegisterUserRequest) Validate() validator.ValidationErrors {

	v := validator.New()

	validator.Required(
		v,
		"zitadel_user_id",
		r.ZitadelUserID,
	)

	validator.Required(
		v,
		"email",
		r.Email,
	)

	validator.Email(
		v,
		"email",
		r.Email,
	)

	if v.Valid() {
		return nil
	}

	return v.Errors
}
