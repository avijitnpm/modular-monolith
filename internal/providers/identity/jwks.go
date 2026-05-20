package identity

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`

	Kty string `json:"kty"`

	Alg string `json:"alg"`

	Use string `json:"use"`
}
