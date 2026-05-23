package config

import (
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

func Load() (*Config, error) {
	loadEnvFile()

	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(s string) string {
		return s
	}), nil)

	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			Name: k.String("APP_NAME"),
			Env:  k.String("APP_ENV"),
		},
		Server: ServerConfig{
			Port: k.String("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			URL: k.String("DATABASE_URL"),
		},
		Auth: AuthConfig{
			ZitadelIssuer: k.String("ZITADEL_ISSUER"),
			ZitadelAPIURL: k.String("ZITADEL_API_URL"),
		},
		Payments: PaymentConfig{
			DodoAPIKey:    k.String("DODO_API_KEY"),
			WebhookSecret: k.String("DODO_WEBHOOK_SECRET"),
		},
		OTEL: OTELConfig{
			Enabled:     k.Bool("OTEL_ENABLED"),
			ServiceName: k.String("OTEL_SERVICE_NAME"),
			Endpoint:    k.String("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Insecure:    k.Bool("OTEL_EXPORTER_OTLP_INSECURE"),
		},
	}

	if cfg.OTEL.ServiceName == "" {
		cfg.OTEL.ServiceName = cfg.App.Name
	}

	err = validate(cfg)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}
