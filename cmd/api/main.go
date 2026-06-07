package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ekideno/zaplink/internal/config"
	"github.com/ekideno/zaplink/internal/db"
	apphttp "github.com/ekideno/zaplink/internal/http"
	"github.com/ekideno/zaplink/internal/logger"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := logger.NewDefault()
	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config",
			slog.String("error", err.Error()),
		)
		return 1
	}

	log, closeLog, err := logger.New(logger.Config{
		Env:      cfg.ENV,
		LogFile:  cfg.Log.File,
		LogLevel: cfg.Log.Level,
	})
	if err != nil {
		log.Error("failed to initialize logger",
			slog.String("error", err.Error()),
		)
		return 1
	}
	defer func() {
		if err := closeLog(); err != nil {
			log.Error("failed to close logger",
				slog.String("error", err.Error()),
			)
		}
	}()

	database, err := db.NewPostgres(context.Background(), cfg.DB.URL)
	if err != nil {
		log.Error("failed to initialize database",
			slog.String("error", err.Error()),
		)
		return 1
	}
	defer database.Close()

	r := apphttp.NewRouter(log)
	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server started",
			slog.String("port", cfg.HTTPPort),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Error("failed to start server",
			slog.String("error", err.Error()),
		)
		return 1
	case <-stop:
	}

	log.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown server",
			slog.String("error", err.Error()),
		)
		return 1
	}

	log.Info("Server stopped gracefully")
	return 0
}
