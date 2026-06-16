package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/ekideno/zaplink/internal/service"
)

func TestCreateLink_Success(t *testing.T) {
	repo := &mockLinkRepo{
		createFn: func(ctx context.Context, link *model.Link) error {
			return nil
		},
	}
	clicks := &mockClickRepo{}

	svc := service.NewLinkService(repo, clicks, nil, 0, slog.Default())
	link, err := svc.CreateLink(context.Background(), "https://google.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link.ShortCode == "" {
		t.Error("expected short code to be set")
	}
}

func TestCreateLink_InvalidURL(t *testing.T) {
	svc := service.NewLinkService(&mockLinkRepo{}, &mockClickRepo{}, nil, 0, slog.Default())

	_, err := svc.CreateLink(context.Background(), "not-a-url")
	if !errors.Is(err, service.ErrInvalidURL) {
		t.Errorf("expected ErrInvalidURL, got %v", err)
	}
}

func TestGetByShortCode_NotFound(t *testing.T) {
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return nil, repository.ErrNotFound
		},
	}
	clicks := &mockClickRepo{}

	svc := service.NewLinkService(repo, clicks, nil, 0, slog.Default())
	_, err := svc.GetByShortCode(context.Background(), "abc123")

	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByShortCode_Inactive(t *testing.T) {
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{IsActive: false}, nil
		},
	}
	clicks := &mockClickRepo{}

	svc := service.NewLinkService(repo, clicks, nil, 0, slog.Default())
	_, err := svc.GetByShortCode(context.Background(), "abc123")

	if !errors.Is(err, service.ErrInactiveLink) {
		t.Errorf("expected ErrInactiveLink, got %v", err)
	}
}
