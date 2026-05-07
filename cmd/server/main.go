package main

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/config"
	"github.com/avijitnpm/modular-monolith/internal/router"
	applogger "github.com/avijitnpm/modular-monolith/pkg/logger"
)

func main() {

	cfg, err := config.Load()

	if err != nil {
		panic(err)
	}

	logger := applogger.New(cfg)

	logger.Info(
		"application starting",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
	)

	r := router.New(logger)

	logger.Info(
		"server running",
		"port", cfg.Server.Port,
	)

	err = http.ListenAndServe(":"+cfg.Server.Port, r)

	if err != nil {
		logger.Error(
			"server failed",
			"error", err,
		)
	}
}
