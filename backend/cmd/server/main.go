package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"ddz/backend/internal/app"
)

func main() {
	cfg := app.LoadConfig()
	server := app.NewHTTPServer(cfg)

	slog.Info("starting ddz backend", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
