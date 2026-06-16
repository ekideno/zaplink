package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ekideno/zaplink/internal/cache"
	"github.com/ekideno/zaplink/internal/config"
	"github.com/ekideno/zaplink/internal/db"
	apphttp "github.com/ekideno/zaplink/internal/http"
	"github.com/ekideno/zaplink/internal/http/handler"
	"github.com/ekideno/zaplink/internal/logger"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/ekideno/zaplink/internal/service"
	"github.com/redis/go-redis/v9"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(logger.Config{
		Env:      cfg.ENV,
		LogLevel: cfg.Log.Level,
	})

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	database, err := db.NewPostgres(connectCtx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer database.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(connectCtx).Err(); err != nil {
		_ = redisClient.Close()
		return fmt.Errorf("ping redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("close redis", slog.Any("error", err))
		}
	}()

	linkRepo := repository.NewLinkPostgres(database)
	clickRepo := repository.NewClickPostgres(database)
	linkCache := cache.NewRedisCache(redisClient)
	linkService := service.NewLinkService(linkRepo, clickRepo, linkCache, cfg.Redis.TTL)
	linkHandler := handler.NewLinkHandler(linkService, log)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      apphttp.NewRouter(log, linkHandler),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server started", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		stop()
	}

	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("stopped gracefully")
	return nil
}
