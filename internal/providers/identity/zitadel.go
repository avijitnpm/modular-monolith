package identity

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret-development-key")

type ZitadelProvider struct{}

func NewZitadelProvider() *ZitadelProvider {
	return &ZitadelProvider{}
}

func GenerateToken(
	userID string,
	organizationID string,
	email string,
) (string, error) {

	claims := Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		Email:          email,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(24 * time.Hour),
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(jwtSecret)
}

func (z *ZitadelProvider) ValidateToken(
	ctx context.Context,
	tokenString string,
) (*Claims, error) {

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
