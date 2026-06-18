package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ekideno/zaplink/internal/cache"
	"github.com/ekideno/zaplink/internal/repository"
)

type HealthHandler struct {
	log        *slog.Logger
	repository repository.LinkRepository
	cache      cache.LinkCache
}

func NewHealthHandler(log *slog.Logger, repo repository.LinkRepository, linkCache cache.LinkCache) *HealthHandler {
	return &HealthHandler{
		log:        log,
		repository: repo,
		cache:      linkCache,
	}
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
	Duration string            `json:"duration"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	overallHealthy := true

	if err := h.repository.Ping(ctx); err != nil {
		checks["postgres"] = "unhealthy"
		overallHealthy = false
		h.log.Error("postgres health check failed", slog.Any("error", err))
	} else {
		checks["postgres"] = "healthy"
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["redis"] = "degraded"
			h.log.Warn("redis health check failed", slog.Any("error", err))
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "disabled"
	}

	status := "healthy"
	statusCode := http.StatusOK

	if !overallHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:   status,
		Checks:   checks,
		Duration: time.Since(start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
