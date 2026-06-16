package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ekideno/zaplink/internal/apperror"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("json encode failed", slog.Int("status", status), slog.String("error", err.Error()))
	}
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperror.From(err)
	if appErr == nil {
		if log != nil {
			log.Error("request failed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", http.StatusInternalServerError),
				slog.String("error", err.Error()),
			)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": http.StatusText(http.StatusInternalServerError)})
		return
	}

	if log != nil {
		log.Error("request failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", appErr.Status),
			slog.String("code", appErr.Code),
			slog.String("error", err.Error()),
		)
	}

	writeJSON(w, appErr.Status, map[string]string{"error": appErr.Message})
}
