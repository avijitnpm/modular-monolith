package identity

type OIDCConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
}
