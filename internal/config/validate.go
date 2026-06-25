package config

import (
	"errors"
	"strings"
)

func validate(cfg *Config) error {
	if cfg.Server.Port == "" {
		return errors.New("SERVER_PORT is required")
	}

	if cfg.Database.URL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if cfg.App.Env == "" {
		return errors.New("APP_ENV is required")
	}

	if cfg.Auth.OIDCIssuer == "" {
		return errors.New("OIDC_ISSUER is required")
	}

	if cfg.Auth.OIDCAudience == "" {
		return errors.New("OIDC_AUDIENCE is required")
	}

	if cfg.Auth.OIDCClientID == "" {
		return errors.New("OIDC_CLIENT_ID is required")
	}

	if cfg.Auth.OIDCRedirectURL == "" {
		return errors.New("OIDC_REDIRECT_URL is required")
	}

	if cfg.Auth.SessionSecret == "" {
		return errors.New("SESSION_SECRET is required")
	}

	if len(cfg.Auth.SessionSecret) < 32 {
		return errors.New("SESSION_SECRET must be at least 32 characters")
	}

	if cfg.App.Env == "development" && cfg.Auth.DevTokenSecret == "" {
		return errors.New("DEV_TOKEN_SECRET is required in development")
	}

	if cfg.Payments.DodoAPIKey == "" {
		return errors.New("DODO_API_KEY is required")
	}

	if cfg.Payments.WebhookSecret == "" {
		return errors.New("DODO_WEBHOOK_SECRET is required")
	}

	if cfg.App.Env == "production" {
		if err := validateProduction(cfg); err != nil {
			return err
		}
	}

	return nil
}

func validateProduction(cfg *Config) error {
	if cfg.App.CORSOrigin == "" {
		return errors.New("CORS_ORIGIN is required in production")
	}

	if cfg.Metrics.Token == "" {
		return errors.New("METRICS_TOKEN is required in production")
	}

	if cfg.Payments.DodoBaseURL == "" {
		return errors.New("DODO_BASE_URL is required in production")
	}

	// Reject placeholder secrets
	if strings.HasPrefix(cfg.Auth.SessionSecret, "change-me") {
		return errors.New("SESSION_SECRET must not use the default placeholder in production")
	}

	// DEV_TOKEN_SECRET must not be set in production
	if cfg.Auth.DevTokenSecret != "" {
		return errors.New("DEV_TOKEN_SECRET must not be set in production")
	}

	// OIDC redirect must use HTTPS
	if !strings.HasPrefix(cfg.Auth.OIDCRedirectURL, "https://") {
		return errors.New("OIDC_REDIRECT_URL must use https:// in production")
	}

	// CORS origin must use HTTPS
	if !strings.HasPrefix(cfg.App.CORSOrigin, "https://") {
		return errors.New("CORS_ORIGIN must use https:// in production")
	}

	// Warn about insecure DB connections
	if strings.Contains(cfg.Database.URL, "sslmode=disable") {
		return errors.New("DATABASE_URL must not use sslmode=disable in production")
	}

	// OTEL should not be insecure in production
	if cfg.OTEL.Enabled && cfg.OTEL.Insecure {
		return errors.New("OTEL_EXPORTER_OTLP_INSECURE must be false in production")
	}

	return nil
}
