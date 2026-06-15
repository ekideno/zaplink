package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/ekideno/zaplink/internal/service"
	"github.com/go-chi/chi/v5"
)

type LinkService interface {
	CreateLink(ctx context.Context, url string) (*model.Link, error)
	GetByShortCode(ctx context.Context, shortCode string) (*model.Link, error)
	GetLinkInfo(ctx context.Context, shortCode string) (*service.LinkInfo, error)
	TrackClick(ctx context.Context, linkID int64, userAgent, remoteAddr string) error
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

type linkInfoResponse struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	ClicksCount int64  `json:"clicks_count"`
}

type LinkHandler struct {
	service LinkService
	log     *slog.Logger
}

func NewLinkHandler(svc LinkService, log *slog.Logger) *LinkHandler {
	return &LinkHandler{service: svc, log: log}
}

func (h *LinkHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}

	link, err := h.service.CreateLink(r.Context(), req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createLinkResponse{
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
	})
}

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "short_code")

	link, err := h.service.GetByShortCode(r.Context(), shortCode)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "link not found"})
		case errors.Is(err, service.ErrInactiveLink):
			writeJSON(w, http.StatusGone, map[string]string{"error": "link is inactive"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	go h.service.TrackClick(context.Background(), link.ID, r.UserAgent(), r.RemoteAddr)

	http.Redirect(w, r, link.OriginalURL, http.StatusFound)
}

func (h *LinkHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "short_code")

	info, err := h.service.GetLinkInfo(r.Context(), shortCode)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "link not found"})
		case errors.Is(err, service.ErrInactiveLink):
			writeJSON(w, http.StatusGone, map[string]string{"error": "link is inactive"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, linkInfoResponse{
		ShortCode:   info.Link.ShortCode,
		OriginalURL: info.Link.OriginalURL,
		IsActive:    info.Link.IsActive,
		CreatedAt:   info.Link.CreatedAt.Format(time.RFC3339),
		ClicksCount: info.ClicksCount,
	})
}
