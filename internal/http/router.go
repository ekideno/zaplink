package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ekideno/zaplink/internal/http/handler"
	appmiddleware "github.com/ekideno/zaplink/internal/middleware"
)

func NewRouter(log *slog.Logger, linkHandler *handler.LinkHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(appmiddleware.RequestLogger(log))
	r.Use(appmiddleware.PrometheusMetrics)

	r.Mount("/metrics", promhttp.Handler())

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Post("/links", linkHandler.CreateLink)
	r.Get("/links/{short_code}", linkHandler.GetInfo)
	r.Get("/{short_code}", linkHandler.Redirect)

	return r
}
