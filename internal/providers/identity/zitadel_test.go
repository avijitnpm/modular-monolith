package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken(t *testing.T) {
	key := newTestRSAKey(t)
	otherKey := newTestRSAKey(t)
	audience := "test-audience"
	organizationID := "org-123"
	issuer := "http://issuer.example"

	client, jwksRequests := newTestOIDCClient(
		t,
		issuer,
		&key.PublicKey,
		"test-key",
	)

	provider := newTestProvider(
		client,
		issuer,
		audience,
	)

	token := newTestToken(
		t,
		key,
		"test-key",
		issuer,
		audience,
		time.Now().Add(time.Hour),
		organizationID,
	)

	claims, err := provider.ValidateToken(
		context.Background(),
		token,
	)

	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Fatalf("expected user id from subject, got %q", claims.UserID)
	}

	if claims.OrganizationID != organizationID {
		t.Fatalf("expected organization id %q, got %q", organizationID, claims.OrganizationID)
	}

	if claims.Email != "test@example.com" {
		t.Fatalf("expected email claim, got %q", claims.Email)
	}

	if got := atomic.LoadInt32(jwksRequests); got != 1 {
		t.Fatalf("expected one jwks fetch, got %d", got)
	}

	_, err = provider.ValidateToken(
		context.Background(),
		token,
	)

	if err != nil {
		t.Fatalf("expected cached key to validate token, got error: %v", err)
	}

	if got := atomic.LoadInt32(jwksRequests); got != 1 {
		t.Fatalf("expected cached jwks to avoid refetch, got %d fetches", got)
	}

	tests := []struct {
		name  string
		token string
	}{
		{
			name: "expired token",
			token: newTestToken(
				t,
				key,
				"test-key",
				issuer,
				audience,
				time.Now().Add(-time.Hour),
				organizationID,
			),
		},
		{
			name: "wrong issuer",
			token: newTestToken(
				t,
				key,
				"test-key",
				"http://wrong-issuer.example",
				audience,
				time.Now().Add(time.Hour),
				organizationID,
			),
		},
		{
			name: "wrong audience",
			token: newTestToken(
				t,
				key,
				"test-key",
				issuer,
				"wrong-audience",
				time.Now().Add(time.Hour),
				organizationID,
			),
		},
		{
			name:  "malformed token",
			token: "not-a-jwt",
		},
		{
			name: "invalid signature",
			token: newTestToken(
				t,
				otherKey,
				"test-key",
				issuer,
				audience,
				time.Now().Add(time.Hour),
				organizationID,
			),
		},
		{
			name: "hs256 token",
			token: newTestHS256Token(
				t,
				issuer,
				audience,
				organizationID,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.ValidateToken(
				context.Background(),
				tt.token,
			)

			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateTokenRejectsOIDCIssuerMismatch(t *testing.T) {
	key := newTestRSAKey(t)
	issuer := "http://issuer.example"

	client, _ := newTestOIDCClientWithDiscoveryIssuer(
		t,
		issuer,
		"http://different-issuer.example",
		&key.PublicKey,
		"test-key",
	)

	provider := NewZitadelProvider(
		OIDCConfig{
			Issuer:   issuer,
			Audience: "test-audience",
		},
	)
	provider.client = client
	provider.jwks.client = client

	token := newTestToken(
		t,
		key,
		"test-key",
		issuer,
		"test-audience",
		time.Now().Add(time.Hour),
		"org-123",
	)

	_, err := provider.ValidateToken(
		context.Background(),
		token,
	)

	if err == nil {
		t.Fatal("expected issuer mismatch error")
	}
}

func TestJWKRSAPublicKey(t *testing.T) {
	key := newTestRSAKey(t)

	jwk := newTestJWK(
		&key.PublicKey,
		"test-key",
	)

	publicKey, err := jwk.rsaPublicKey()

	if err != nil {
		t.Fatalf("expected valid rsa public key, got error: %v", err)
	}

	if publicKey.N.Cmp(key.PublicKey.N) != 0 {
		t.Fatal("expected modulus to match")
	}

	if publicKey.E != key.PublicKey.E {
		t.Fatal("expected exponent to match")
	}

	_, err = JWK{
		Kid: "bad-key",
		Kty: "EC",
		Alg: "ES256",
		Use: "sig",
	}.rsaPublicKey()

	if err == nil {
		t.Fatal("expected unsupported jwk error")
	}
}

func newTestProvider(
	client *http.Client,
	issuer string,
	audience string,
) *ZitadelProvider {
	provider := NewZitadelProvider(
		OIDCConfig{
			Issuer:   issuer,
			Audience: audience,
		},
	)
	provider.client = client
	provider.jwks.client = client

	return provider
}

func newTestOIDCClient(
	t *testing.T,
	issuer string,
	key *rsa.PublicKey,
	kid string,
) (*http.Client, *int32) {
	t.Helper()

	return newTestOIDCClientWithDiscoveryIssuer(
		t,
		issuer,
		issuer,
		key,
		kid,
	)
}

func newTestOIDCClientWithDiscoveryIssuer(
	t *testing.T,
	issuer string,
	discoveryIssuer string,
	key *rsa.PublicKey,
	kid string,
) (*http.Client, *int32) {
	t.Helper()

	var jwksRequests int32
	jwksURI := issuer + "/keys"

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case issuer + "/.well-known/openid-configuration":
				return jsonResponse(
					t,
					OIDCDiscovery{
						Issuer:  discoveryIssuer,
						JWKSURI: jwksURI,
					},
				), nil
			case jwksURI:
				atomic.AddInt32(
					&jwksRequests,
					1,
				)

				return jsonResponse(
					t,
					JWKS{
						Keys: []JWK{
							newTestJWK(
								key,
								kid,
							),
						},
					},
				), nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     http.Header{},
					Request:    r,
				}, nil
			}
		}),
	}

	return client, &jwksRequests
}

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(
		rand.Reader,
		2048,
	)

	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	return key
}

func newTestJWK(
	key *rsa.PublicKey,
	kid string,
) JWK {
	exponent := big.NewInt(
		int64(key.E),
	).Bytes()

	return JWK{
		Kid: kid,
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func newTestToken(
	t *testing.T,
	key *rsa.PrivateKey,
	kid string,
	issuer string,
	audience string,
	expiresAt time.Time,
	organizationID string,
) string {
	t.Helper()

	claims := Claims{
		OrganizationID: organizationID,
		Email:          "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  issuer,
			Subject: "user-123",
			Audience: jwt.ClaimStrings{
				audience,
			},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		claims,
	)
	token.Header["kid"] = kid

	signedToken, err := token.SignedString(key)

	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signedToken
}

func newTestHS256Token(
	t *testing.T,
	issuer string,
	audience string,
	organizationID string,
) string {
	t.Helper()

	claims := Claims{
		OrganizationID: organizationID,
		Email:          "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  issuer,
			Subject: "user-123",
			Audience: jwt.ClaimStrings{
				audience,
			},
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(time.Hour),
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)
	token.Header["kid"] = "test-key"

	signedToken, err := token.SignedString(
		[]byte("test-secret"),
	)

	if err != nil {
		t.Fatalf("sign hs256 token: %v", err)
	}

	return signedToken
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(
	t *testing.T,
	value interface{},
) *http.Response {
	t.Helper()

	var body strings.Builder

	if err := json.NewEncoder(&body).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body.String())),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}
}
