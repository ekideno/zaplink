package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string
	ENV      string

	DB DatabaseConfig

	Log LogConfig
}

type DatabaseConfig struct {
	URL string
}

type LogConfig struct {
	Level slog.Level
	File  string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		ENV:      getEnv("ENV", "dev"),

		DB: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},

		Log: LogConfig{
			Level: parseLogLevel(getEnv("LOG_LEVEL", "info")),
			File:  getEnv("LOG_FILE", "./logs/app.log"),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
func (c Config) validate() error {
	if c.DB.URL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if c.HTTPPort == "" {
		return errors.New("HTTP_PORT is empty")
	}

	return nil
}
func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
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
