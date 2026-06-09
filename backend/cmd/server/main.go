package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"ddz/backend/internal/app"
)

func main() {
	cfg := app.LoadConfig()
	slog.SetDefault(appLogger(cfg))
	server := app.NewHTTPServer(cfg)

	slog.Info("starting ddz backend", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func appLogger(cfg app.Config) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	}))
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
