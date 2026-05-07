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

	return nil
}
