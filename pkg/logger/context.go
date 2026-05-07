package logger

import (
	"context"
	"log/slog"
)

type contextKey string

const LoggerKey contextKey = "logger"

func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, log)
}

func FromContext(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(LoggerKey).(*slog.Logger)

	if !ok {
		return slog.Default()
	}

	return log
}
