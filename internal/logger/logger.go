package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	Env      string // "dev" | "prod"
	LogLevel slog.Level
}

func New(cfg Config) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}

	switch cfg.Env {
	case "prod":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		slog.String("service", "zaplink"),
		slog.String("env", cfg.Env),
	)
	slog.SetDefault(logger)

	return logger
}

func NewDefault() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
}
