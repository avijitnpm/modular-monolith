package auth

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID string `json:"user_id"`

	OrganizationID string `json:"organization_id"`

	Email string `json:"email"`

	jwt.RegisteredClaims
}
