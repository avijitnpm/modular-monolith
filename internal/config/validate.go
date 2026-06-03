package config

import (
	"errors"
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

	return nil
}
