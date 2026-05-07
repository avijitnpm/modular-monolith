package logger

import (
	"log/slog"
	"os"

	"github.com/avijitnpm/modular-monolith/internal/config"
)

func New(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	if cfg.App.Env == EnvDevelopment {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}
