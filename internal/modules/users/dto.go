package users

type RegisterUserRequest struct {
	ZitadelUserID string `json:"zitadel_user_id"`
	Email         string `json:"email"`
}
