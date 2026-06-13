package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ekideno/zaplink/internal/http/handler"
	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/service"
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
