package app

import (
	"context"
	"time"
)

func (a *App) Shutdown() {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if a.HTTPServer != nil {
		if err := a.HTTPServer.Shutdown(ctx); err != nil {
			a.Logger.Error(
				"http server shutdown failed",
				"error", err,
			)
		}
	}

	if a.ShutdownOTEL != nil {
		if err := a.ShutdownOTEL(ctx); err != nil {
			a.Logger.Error(
				"otel shutdown failed",
				"error", err,
			)
		}
	}

	if a.DB != nil {
		a.DB.Close()
	}

	a.Logger.Info("application shutdown complete")
}
