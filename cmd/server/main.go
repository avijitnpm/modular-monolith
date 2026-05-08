package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/avijitnpm/modular-monolith/internal/app"
)

func main() {

	application, err := app.New()

	if err != nil {
		panic(err)
	}

	go func() {
		err := application.Start()

		if err != nil {
			application.Logger.Error(
				"server failed",
				"error", err,
			)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-stop

	application.Logger.Info("shutdown signal received")

	application.Shutdown()
}
