package app

func (a *App) Shutdown() {

	if a.DB != nil {
		a.DB.Close()
	}

	a.Logger.Info("application shutdown complete")
}
