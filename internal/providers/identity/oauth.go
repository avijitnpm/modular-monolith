package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type AuthorizationRequest struct {
	State         string
	Nonce         string
	CodeChallenge string
	Scope         string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (z *ZitadelProvider) AuthorizationURL(
	ctx context.Context,
	request AuthorizationRequest,
) (string, error) {
	discovery, err := z.getDiscovery(ctx)

	if err != nil {
		return "", fmt.Errorf("oidc discovery failed: %w", err)
	}

	if discovery.AuthorizationEndpoint == "" {
		return "", fmt.Errorf("oidc discovery missing authorization_endpoint")
	}

	values := url.Values{}
	values.Set("client_id", z.config.ClientID)
	values.Set("redirect_uri", z.config.RedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", request.Scope)
	values.Set("state", request.State)
	values.Set("nonce", request.Nonce)
	values.Set("code_challenge", request.CodeChallenge)
	values.Set("code_challenge_method", "S256")

	return discovery.AuthorizationEndpoint + "?" + values.Encode(), nil
}

func (z *ZitadelProvider) ExchangeCode(
	ctx context.Context,
	code string,
	codeVerifier string,
) (*TokenResponse, error) {
	discovery, err := z.getDiscovery(ctx)

	if err != nil {
		return nil, fmt.Errorf("oidc discovery failed: %w", err)
	}

	if discovery.TokenEndpoint == "" {
		return nil, fmt.Errorf("oidc discovery missing token_endpoint")
	}

	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", z.config.RedirectURL)
	values.Set("client_id", z.config.ClientID)
	values.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		discovery.TokenEndpoint,
		strings.NewReader(values.Encode()),
	)

	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	req.Header.Set(
		"Accept",
		"application/json",
	)

	resp, err := z.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange code: unexpected status %d", resp.StatusCode)
	}

	var token TokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if token.IDToken == "" {
		return nil, fmt.Errorf("token response missing id_token")
	}

	return &token, nil
}
