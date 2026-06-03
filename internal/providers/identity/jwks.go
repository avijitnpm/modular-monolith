package identity

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

const jwksCacheTTL = 5 * time.Minute

var (
	errJWKSKeyNotFound = errors.New("jwks key not found")
	errInvalidJWK      = errors.New("invalid jwk")
)

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`

	Kty string `json:"kty"`

	Alg string `json:"alg"`

	Use string `json:"use"`

	N string `json:"n"`

	E string `json:"e"`
}

type jwksCache struct {
	client *http.Client
	ttl    time.Duration

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func newJWKSCache(client *http.Client) *jwksCache {
	return &jwksCache{
		client: client,
		ttl:    jwksCacheTTL,
		keys:   map[string]*rsa.PublicKey{},
	}
}

func (c *jwksCache) key(
	ctx context.Context,
	jwksURI string,
	kid string,
) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if now.Before(c.expiresAt) {
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}

	if err := c.refresh(ctx, jwksURI); err != nil {
		return nil, err
	}

	key, ok := c.keys[kid]

	if !ok {
		return nil, errJWKSKeyNotFound
	}

	return key, nil
}

func (c *jwksCache) refresh(
	ctx context.Context,
	jwksURI string,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		jwksURI,
		nil,
	)

	if err != nil {
		return fmt.Errorf("create jwks request: %w", err)
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var jwks JWKS

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := map[string]*rsa.PublicKey{}

	for _, jwk := range jwks.Keys {
		key, err := jwk.rsaPublicKey()

		if err != nil {
			continue
		}

		keys[jwk.Kid] = key
	}

	c.keys = keys
	c.expiresAt = time.Now().Add(c.ttl)

	return nil
}

func (j JWK) rsaPublicKey() (*rsa.PublicKey, error) {
	if j.Kid == "" ||
		j.Kty != "RSA" ||
		(j.Use != "" && j.Use != "sig") ||
		(j.Alg != "" && j.Alg != "RS256") {
		return nil, errInvalidJWK
	}

	modulus, err := base64.RawURLEncoding.DecodeString(j.N)

	if err != nil {
		return nil, fmt.Errorf("%w: modulus", errInvalidJWK)
	}

	exponent, err := base64.RawURLEncoding.DecodeString(j.E)

	if err != nil {
		return nil, fmt.Errorf("%w: exponent", errInvalidJWK)
	}

	e := new(big.Int).SetBytes(exponent).Int64()

	if e <= 0 {
		return nil, fmt.Errorf("%w: exponent", errInvalidJWK)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(e),
	}, nil
}
