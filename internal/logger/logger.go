package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	Env      string // "dev" | "prod"
	LogFile  string
	LogLevel slog.Level
}

func NewDefault() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
}

func New(cfg Config) (*slog.Logger, func() error, error) {
	err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(
		cfg.LogFile,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	var handler slog.Handler

	switch cfg.Env {

	case "prod":
		consoleHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		})

		fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		})

		handler = newMultiHandler(consoleHandler, fileHandler)

	default:
		consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		})

		fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		})

		handler = newMultiHandler(consoleHandler, fileHandler)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, file.Close, nil
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		hs[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: hs}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		hs[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: hs}
}
