package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string
	ENV      string

	DB    DatabaseConfig
	Redis RedisConfig

	Log LogConfig
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
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
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			TTL:      time.Duration(getEnvInt("REDIS_TTL_SECONDS", 3600)) * time.Second,
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

	if c.Redis.Addr == "" {
		return errors.New("REDIS_ADDR is empty")
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

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}

	var parsed int
	_, err := fmt.Sscanf(val, "%d", &parsed)
	if err != nil {
		return fallback
	}
	return parsed
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
