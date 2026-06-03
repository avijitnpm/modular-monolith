package identity

type OIDCConfig struct {
	Issuer      string
	Audience    string
	JWKSURL     string
	ClientID    string
	RedirectURL string
}
