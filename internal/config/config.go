package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"log/slog"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string
	DBURL    string

	ENV string

	LogLevel slog.Level
	LogFile  string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("warn: could not load .env file: %v", err)
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required but not set")
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info"
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "./logs/app.log"
	}

	return Config{
		HTTPPort: port,
		DBURL:    dbURL,
		ENV:      env,
		LogLevel: parseLogLevel(logLevelStr),
		LogFile:  logFile,
	}, nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
