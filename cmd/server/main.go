package main

import (
	"log"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/config"
	applogger "github.com/avijitnpm/modular-monolith/pkg/logger"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	logger := applogger.New(cfg)

	logger.Info(
		"application starting",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
	)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	logger.Info(
		"server running",
		"port", cfg.Server.Port,
	)

	err = http.ListenAndServe(":"+cfg.Server.Port, nil)

	if err != nil {
		logger.Error(
			"server failed",
			"error", err,
		)
	}
}
