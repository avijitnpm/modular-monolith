package config

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Payments PaymentConfig
	OTEL     OTELConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL string
}

type AuthConfig struct {
	ZitadelIssuer string
	ZitadelAPIURL string
}

type PaymentConfig struct {
	DodoAPIKey    string
	WebhookSecret string
}

type OTELConfig struct {
	Endpoint    string
	ServiceName string
}
