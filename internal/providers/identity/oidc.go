package identity

type OIDCDiscovery struct {
	Issuer string `json:"issuer"`

	JWKSURI string `json:"jwks_uri"`

	AuthorizationEndpoint string `json:"authorization_endpoint"`

	TokenEndpoint string `json:"token_endpoint"`
}
