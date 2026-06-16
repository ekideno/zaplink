package service_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/ekideno/zaplink/internal/apperror"
	"github.com/ekideno/zaplink/internal/cache"
	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/ekideno/zaplink/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestCreateLink_RetriesOnUniqueViolation(t *testing.T) {
	attempts := 0
	repo := &mockLinkRepo{
		createFn: func(ctx context.Context, link *model.Link) error {
			attempts++
			if attempts == 1 {
				return &pgconn.PgError{Code: "23505"}
			}
			return nil
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, nil, 0, slog.Default())
	link, err := svc.CreateLink(context.Background(), "https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if link == nil || link.ShortCode == "" {
		t.Fatal("expected link with short code")
	}
}

func TestGetLinkInfo_Success(t *testing.T) {
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 42, ShortCode: shortCode, OriginalURL: "https://example.com", IsActive: true}, nil
		},
	}
	clicks := &mockClickRepo{
		countFn: func(ctx context.Context, linkID int64) (int64, error) {
			if linkID != 42 {
				t.Fatalf("expected link id 42, got %d", linkID)
			}
			return 17, nil
		},
	}

	svc := service.NewLinkService(repo, clicks, nil, 0, slog.Default())
	info, err := svc.GetLinkInfo(context.Background(), "abc123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.ClicksCount != 17 {
		t.Fatalf("expected 17 clicks, got %d", info.ClicksCount)
	}
}

func TestGetLinkInfo_CountError(t *testing.T) {
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 42, ShortCode: shortCode, OriginalURL: "https://example.com", IsActive: true}, nil
		},
	}
	clicks := &mockClickRepo{
		countFn: func(ctx context.Context, linkID int64) (int64, error) {
			return 0, errors.New("count failed")
		},
	}

	svc := service.NewLinkService(repo, clicks, nil, 0, slog.Default())
	_, err := svc.GetLinkInfo(context.Background(), "abc123")

	if err == nil {
		t.Fatal("expected error")
	}
	if apperror.Status(err) != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", apperror.Status(err))
	}
}

func TestTrackClick_ParsesRemoteAddrAndUserAgent(t *testing.T) {
	var gotClick *model.Click
	clicks := &mockClickRepo{
		createFn: func(ctx context.Context, click *model.Click) error {
			gotClick = click
			return nil
		},
	}

	svc := service.NewLinkService(&mockLinkRepo{}, clicks, nil, 0, slog.Default())
	err := svc.TrackClick(context.Background(), 99, "Mozilla/5.0", "203.0.113.10:1234")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotClick == nil {
		t.Fatal("expected click to be created")
	}
	if gotClick.LinkID != 99 {
		t.Fatalf("expected link id 99, got %d", gotClick.LinkID)
	}
	if gotClick.UserAgent == nil || *gotClick.UserAgent != "Mozilla/5.0" {
		t.Fatalf("expected user agent to be set, got %#v", gotClick.UserAgent)
	}
	if gotClick.IPAddress == nil || *gotClick.IPAddress != "203.0.113.10" {
		t.Fatalf("expected ip address 203.0.113.10, got %#v", gotClick.IPAddress)
	}
}

func TestTrackClick_ErrorWrapped(t *testing.T) {
	clicks := &mockClickRepo{
		createFn: func(ctx context.Context, click *model.Click) error {
			return errors.New("db down")
		},
	}

	svc := service.NewLinkService(&mockLinkRepo{}, clicks, nil, 0, slog.Default())
	err := svc.TrackClick(context.Background(), 99, "", "")

	if err == nil {
		t.Fatal("expected error")
	}
	if apperror.Status(err) != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", apperror.Status(err))
	}
}

func TestCreateLink_RetriesStopAfterThreeAttempts(t *testing.T) {
	attempts := 0
	repo := &mockLinkRepo{
		createFn: func(ctx context.Context, link *model.Link) error {
			attempts++
			return &pgconn.PgError{Code: "23505"}
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, nil, 0, slog.Default())
	_, err := svc.CreateLink(context.Background(), "https://example.com")

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestCreateLink_SetsActiveFlag(t *testing.T) {
	var got *model.Link
	repo := &mockLinkRepo{
		createFn: func(ctx context.Context, link *model.Link) error {
			got = link
			return nil
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, nil, 0, slog.Default())
	_, err := svc.CreateLink(context.Background(), "https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || !got.IsActive {
		t.Fatal("expected created link to be active")
	}
}

func TestCreateLink_RepoErrorWrapped(t *testing.T) {
	repo := &mockLinkRepo{
		createFn: func(ctx context.Context, link *model.Link) error {
			return errors.New("insert failed")
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, nil, 0, slog.Default())
	_, err := svc.CreateLink(context.Background(), "https://example.com")

	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "create link: insert failed" {
		t.Fatalf("expected wrapped error, got %q", got)
	}
}

func TestGetByShortCode_CacheHit(t *testing.T) {
	cacheHit := &mockLinkCache{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 1, ShortCode: shortCode, OriginalURL: "https://cached.example.com", IsActive: true}, nil
		},
	}
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			t.Fatal("repo should not be called on cache hit")
			return nil, nil
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, cacheHit, time.Minute, slog.Default())
	link, err := svc.GetByShortCode(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link.OriginalURL != "https://cached.example.com" {
		t.Fatalf("expected cached link, got %q", link.OriginalURL)
	}
}

func TestGetByShortCode_CacheMissFallsBackToRepoAndWarmsCache(t *testing.T) {
	var setCalled bool
	cacheMiss := &mockLinkCache{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return nil, cache.ErrCacheMiss
		},
		setLinkFn: func(ctx context.Context, link *model.Link, ttl time.Duration) error {
			setCalled = true
			if ttl != 2*time.Minute {
				t.Fatalf("expected ttl 2m, got %s", ttl)
			}
			if link.ShortCode != "abc123" {
				t.Fatalf("expected cached short code abc123, got %s", link.ShortCode)
			}
			return nil
		},
	}
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 7, ShortCode: shortCode, OriginalURL: "https://repo.example.com", IsActive: true}, nil
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, cacheMiss, 2*time.Minute, slog.Default())
	link, err := svc.GetByShortCode(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link.OriginalURL != "https://repo.example.com" {
		t.Fatalf("expected repo link, got %q", link.OriginalURL)
	}
	if !setCalled {
		t.Fatal("expected cache set to be called")
	}
}

func TestGetByShortCode_CacheErrorStillFallsBack(t *testing.T) {
	cacheLayer := &mockLinkCache{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return nil, errors.New("redis unavailable")
		},
	}
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ID: 7, ShortCode: shortCode, OriginalURL: "https://repo.example.com", IsActive: true}, nil
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, cacheLayer, time.Minute, slog.Default())
	link, err := svc.GetByShortCode(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link.ShortCode != "abc123" {
		t.Fatalf("expected short code abc123, got %s", link.ShortCode)
	}
}

func TestGetByShortCode_CacheInactiveLink(t *testing.T) {
	cacheLayer := &mockLinkCache{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return &model.Link{ShortCode: shortCode, IsActive: false}, nil
		},
	}

	svc := service.NewLinkService(&mockLinkRepo{}, &mockClickRepo{}, cacheLayer, time.Minute, slog.Default())
	_, err := svc.GetByShortCode(context.Background(), "abc123")
	if !errors.Is(err, service.ErrInactiveLink) {
		t.Fatalf("expected ErrInactiveLink, got %v", err)
	}
}

func TestGetByShortCode_RepoTechnicalErrorWrapped(t *testing.T) {
	repo := &mockLinkRepo{
		getByShortCodeFn: func(ctx context.Context, shortCode string) (*model.Link, error) {
			return nil, errors.New("db down")
		},
	}

	svc := service.NewLinkService(repo, &mockClickRepo{}, nil, 0, slog.Default())
	_, err := svc.GetByShortCode(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperror.Status(err) != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", apperror.Status(err))
	}
}
