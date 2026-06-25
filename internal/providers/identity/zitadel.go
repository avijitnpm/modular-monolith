package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	errMissingTokenKeyID  = errors.New("token missing key id")
	errOIDCIssuerMismatch = errors.New("oidc issuer mismatch")
)

type ZitadelProvider struct {
	config OIDCConfig
	client *http.Client
	jwks   *jwksCache

	discoveryMu sync.Mutex
	discovery   *OIDCDiscovery
}

func NewZitadelProvider(
	config OIDCConfig,
) *ZitadelProvider {
	config.Issuer = strings.TrimRight(
		config.Issuer,
		"/",
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	return &ZitadelProvider{
		config: config,
		client: client,
		jwks:   newJWKSCache(client),
	}
}

func GenerateToken(
	secret []byte,
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

	return token.SignedString(secret)
}

func (z *ZitadelProvider) ValidateToken(
	ctx context.Context,
	tokenString string,
) (*Claims, error) {
	discovery, err := z.getDiscovery(ctx)

	if err != nil {
		return nil, fmt.Errorf("oidc discovery failed: %w", err)
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			kid, ok := token.Header["kid"].(string)

			if !ok || kid == "" {
				return nil, errMissingTokenKeyID
			}

			return z.jwks.key(
				ctx,
				discovery.JWKSURI,
				kid,
			)
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodRS256.Alg(),
		}),
		jwt.WithIssuer(z.config.Issuer),
		jwt.WithAudience(z.config.Audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	if claims.UserID == "" {
		claims.UserID = claims.Subject
	}

	return claims, nil
}

func (z *ZitadelProvider) getDiscovery(
	ctx context.Context,
) (*OIDCDiscovery, error) {
	z.discoveryMu.Lock()
	defer z.discoveryMu.Unlock()

	if z.discovery != nil {
		return z.discovery, nil
	}

	discovery, err := discoverOIDC(
		ctx,
		z.client,
		z.config.Issuer,
	)

	if err != nil {
		return nil, err
	}

	if discovery.Issuer != z.config.Issuer {
		return nil, errOIDCIssuerMismatch
	}

	z.discovery = discovery

	return discovery, nil
}
