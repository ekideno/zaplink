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
	log := logger.NewDefault()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	pool := db.NewPostgresPool(cfg.DB.URL)
	defer pool.Close()

	log = logger.New(logger.Config{
		Env:      cfg.ENV,
		LogFile:  cfg.Log.File,
		LogLevel: cfg.Log.Level,
	})
	r := apphttp.NewRouter(log)
	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("server started",
			slog.String("port", cfg.HTTPPort),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}()

	<-stop
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("failed to Shutdown server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
