package app

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/internal/router"
	"github.com/avijitnpm/modular-monolith/internal/service"
	applogger "github.com/avijitnpm/modular-monolith/pkg/logger"
)

func New() (*App, error) {

	cfg, err := config.Load()

	if err != nil {
		return nil, err
	}

	log := applogger.New(cfg)

	db, err := database.New(cfg.Database.URL)

	if err != nil {
		return nil, err
	}

	repo := repository.New(db)

	svc := service.New(repo)

	return &App{
		Config: cfg,
		Logger: log,

		DB: db,

		Repository: repo,
		Service:    svc,
	}, nil
}

func (a *App) Start() error {

	r := router.New(
		a.Logger,
		a.Service,
	)

	a.Logger.Info(
		"server starting",
		"port", a.Config.Server.Port,
	)

	return http.ListenAndServe(
		":"+a.Config.Server.Port,
		r,
	)
}
