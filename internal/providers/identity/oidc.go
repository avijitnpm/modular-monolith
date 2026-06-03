package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OIDCDiscovery struct {
	Issuer string `json:"issuer"`

	JWKSURI string `json:"jwks_uri"`

	AuthorizationEndpoint string `json:"authorization_endpoint"`

	TokenEndpoint string `json:"token_endpoint"`
}

func discoverOIDC(
	ctx context.Context,
	client *http.Client,
	issuer string,
) (*OIDCDiscovery, error) {
	discoveryURL := strings.TrimRight(
		issuer,
		"/",
	) + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		discoveryURL,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("create oidc discovery request: %w", err)
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("fetch oidc discovery: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch oidc discovery: unexpected status %d", resp.StatusCode)
	}

	var discovery OIDCDiscovery

	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("decode oidc discovery: %w", err)
	}

	if discovery.Issuer == "" || discovery.JWKSURI == "" {
		return nil, fmt.Errorf("oidc discovery missing issuer or jwks_uri")
	}

	return &discovery, nil
}
