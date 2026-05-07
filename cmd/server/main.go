package main

import (
	"log"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/config"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting %s in %s mode",
		cfg.App.Name,
		cfg.App.Env,
	)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Printf("server running on %s", cfg.Server.Port)

	http.ListenAndServe(":"+cfg.Server.Port, nil)
}
