package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ekideno/zaplink/internal/http/handler"
	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestCreateLink_Handler_Success(t *testing.T) {
	svc := &mockLinkService{
		createFn: func(ctx context.Context, url string) (*model.Link, error) {
			return &model.Link{ShortCode: "abc123", OriginalURL: url}, nil
		},
	}

	h := handler.NewLinkHandler(svc, slog.Default())
	body := `{"url": "https://google.com"}`

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateLink(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", res.StatusCode)
	}
}

func TestCreateLink_Handler_InvalidURL(t *testing.T) {
	svc := &mockLinkService{
		createFn: func(ctx context.Context, url string) (*model.Link, error) {
			return nil, service.ErrInvalidURL
		},
	}

	h := handler.NewLinkHandler(svc, slog.Default())
	body := `{"url": "not-a-url"}`

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateLink(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestCreateLink_Handler_InvalidJSON(t *testing.T) {
	h := handler.NewLinkHandler(&mockLinkService{}, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"url":`))
	w := httptest.NewRecorder()

	h.CreateLink(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestCreateLink_Handler_MissingURL(t *testing.T) {
	h := handler.NewLinkHandler(&mockLinkService{}, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateLink(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestGetInfo_Handler_Success(t *testing.T) {
	svc := &mockLinkService{
		getLinkInfoFn: func(ctx context.Context, shortCode string) (*service.LinkInfo, error) {
			return &service.LinkInfo{
				Link: &model.Link{
					ShortCode:   shortCode,
					OriginalURL: "https://example.com",
					IsActive:    true,
					CreatedAt:   time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
				},
				ClicksCount: 11,
			}, nil
		},
	}

	h := handler.NewLinkHandler(svc, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/links/abc123", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("short_code", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetInfo(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["short_code"] != "abc123" {
		t.Fatalf("expected short_code abc123, got %v", payload["short_code"])
	}
	if payload["clicks_count"].(float64) != 11 {
		t.Fatalf("expected clicks_count 11, got %v", payload["clicks_count"])
	}
}

func TestRedirect_Handler_Success(t *testing.T) {
	clickDone := make(chan struct{}, 1)
	svc := &mockLinkService{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 7, ShortCode: shortCode, OriginalURL: "https://example.com", IsActive: true}, nil
		},
		recordClickFn: func(ctx context.Context, click *model.Click) error {
			clickDone <- struct{}{}
			return nil
		},
	}

	h := handler.NewLinkHandler(svc, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "192.0.2.1:8080"
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("short_code", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.Redirect(w, req)

	if w.Result().StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Result().StatusCode)
	}
	if got := w.Result().Header.Get("Location"); got != "https://example.com" {
		t.Fatalf("expected redirect to https://example.com, got %q", got)
	}

	select {
	case <-clickDone:
	case <-time.After(2 * time.Second):
		t.Fatal("expected async click tracking")
	}
}
