package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/internal/service"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger

	HTTPServer   *http.Server
	DB           *pgxpool.Pool
	ShutdownOTEL func(context.Context) error

	Repository *repository.Repository
	Service    *service.Service
}
