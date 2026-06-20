package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/internal/router"
	"github.com/avijitnpm/modular-monolith/internal/service"
	applogger "github.com/avijitnpm/modular-monolith/pkg/logger"
	appotel "github.com/avijitnpm/modular-monolith/pkg/otel"
)

func New() (*App, error) {

	cfg, err := config.Load()

	if err != nil {
		return nil, err
	}

	log := applogger.New(cfg)

	shutdownOTEL := appotel.Init(
		context.Background(),
		cfg,
		log,
	)

	db, err := database.New(
		cfg.Database.URL,
		cfg.OTEL.Enabled,
	)

	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if shutdownErr := shutdownOTEL(shutdownCtx); shutdownErr != nil {
			log.Error(
				"otel shutdown failed after app initialization error",
				"error", shutdownErr,
			)
		}

		return nil, err
	}

	repo := repository.New(db)

	svc := service.New(repo, audit.NewService(repo))

	return &App{
		Config: cfg,
		Logger: log,

		DB:           db,
		ShutdownOTEL: shutdownOTEL,

		Repository: repo,
		Service:    svc,
	}, nil
}

func (a *App) Start() error {

	r := router.New(
		a.Config,
		a.Logger,
		a.Service,
	)

	const (
		readHeaderTimeout = 5 * time.Second
		readTimeout       = 15 * time.Second
		writeTimeout      = 30 * time.Second
		idleTimeout       = 60 * time.Second
	)

	a.HTTPServer = &http.Server{
		Addr:              ":" + a.Config.Server.Port,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	a.Logger.Info(
		"server starting",
		"port", a.Config.Server.Port,
	)

	err := a.HTTPServer.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
