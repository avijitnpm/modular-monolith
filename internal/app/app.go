package app

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/internal/service"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger

	DB *pgxpool.Pool

	Repository *repository.Repository
	Service    *service.Service
}
