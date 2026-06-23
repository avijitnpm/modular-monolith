package config

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Payments PaymentConfig
	OTEL     OTELConfig
	Metrics  MetricsConfig
}

type MetricsConfig struct {
	Token string
}

type AppConfig struct {
	Name       string
	Env        string
	CORSOrigin string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL string
}

type AuthConfig struct {
	OIDCIssuer      string
	OIDCAudience    string
	OIDCClientID    string
	OIDCRedirectURL string
	SessionSecret   string
	DevTokenSecret  string
}

type PaymentConfig struct {
	DodoAPIKey    string
	WebhookSecret string
	DodoBaseURL   string
}

type OTELConfig struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	Insecure    bool
}
