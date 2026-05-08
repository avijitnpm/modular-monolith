package app

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avijitnpm/modular-monolith/internal/config"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *pgxpool.Pool
}
