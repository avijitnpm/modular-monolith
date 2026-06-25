package config

import (
	"os"
	"strings"
)

// loadSecrets implements the Docker Secrets file convention.
// For each supported secret, if ENV_VAR_FILE is set (e.g., SESSION_SECRET_FILE=/run/secrets/session_secret),
// the file contents are read and set as the env var value.
// This is backward-compatible: if _FILE is not set, the regular env var is used unchanged.
var secretVars = []string{
	"SESSION_SECRET",
	"DATABASE_URL",
	"POSTGRES_PASSWORD",
	"DODO_API_KEY",
	"DODO_WEBHOOK_SECRET",
	"METRICS_TOKEN",
	"ZO_ROOT_USER_PASSWORD",
	"OTEL_EXPORTER_OTLP_HEADERS",
}

func loadSecrets() {
	for _, name := range secretVars {
		filePath := os.Getenv(name + "_FILE")
		if filePath == "" {
			continue
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		os.Setenv(name, strings.TrimRight(string(data), "\n\r"))
	}
}
