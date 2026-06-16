package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ekideno/zaplink/internal/apperror"
	"github.com/ekideno/zaplink/internal/cache"
	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

type LinkInfo struct {
	Link        *model.Link
	ClicksCount int64
}

type LinkService struct {
	links    repository.LinkRepository
	cache    cache.LinkCache
	clicks   repository.ClickRepository
	cacheTTL time.Duration
}

func NewLinkService(links repository.LinkRepository, clicks repository.ClickRepository, linkCache cache.LinkCache, cacheTTL time.Duration) *LinkService {
	return &LinkService{links: links, clicks: clicks, cache: linkCache, cacheTTL: cacheTTL}
}

func (s *LinkService) CreateLink(ctx context.Context, originalURL string) (*model.Link, error) {
	u, err := url.ParseRequestURI(originalURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, apperror.Wrapf(ErrInvalidURL, http.StatusBadRequest, "invalid_url", "must be http or https")
	}

	const maxRetries = 3
	for range maxRetries {
		shortCode, err := generateShortCode(6)
		if err != nil {
			return nil, fmt.Errorf("generate short code: %w", err)
		}

		link := &model.Link{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
			IsActive:    true,
		}

		if err := s.links.CreateLink(ctx, link); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return nil, fmt.Errorf("create link: %w", err)
		}

		return link, nil
	}

	return nil, fmt.Errorf("failed to generate unique short code after %d attempts", maxRetries)
}

func (s *LinkService) GetByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	if s.cache != nil {
		link, err := s.cache.GetByShortCode(ctx, shortCode)
		if err == nil && link != nil {
			if !link.IsActive {
				return nil, ErrInactiveLink
			}
			return link, nil
		}
	}

	link, err := s.links.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.Wrap(err, http.StatusNotFound, "link_not_found", "link not found")
		}
		return nil, apperror.Wrap(err, http.StatusInternalServerError, "get_link", "get link by short code")
	}

	if !link.IsActive {
		return nil, ErrInactiveLink
	}

	if s.cache != nil {
		_ = s.cache.SetLink(ctx, link, s.cacheTTL)
	}

	return link, nil
}

func (s *LinkService) GetLinkInfo(ctx context.Context, shortCode string) (*LinkInfo, error) {
	link, err := s.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	clicksCount, err := s.clicks.CountClicksByLinkID(ctx, link.ID)
	if err != nil {
		return nil, apperror.Wrap(err, http.StatusInternalServerError, "count_clicks", "count clicks")
	}

	return &LinkInfo{
		Link:        link,
		ClicksCount: clicksCount,
	}, nil
}

func (s *LinkService) TrackClick(ctx context.Context, linkID int64, userAgent, remoteAddr string) error {
	var ipAddress *string
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		ipAddress = &host
	} else if remoteAddr != "" {
		ipAddress = &remoteAddr
	}

	var ua *string
	if userAgent != "" {
		ua = &userAgent
	}

	click := &model.Click{
		LinkID:    linkID,
		UserAgent: ua,
		IPAddress: ipAddress,
	}

	if err := s.clicks.CreateClick(ctx, click); err != nil {
		return apperror.Wrap(err, http.StatusInternalServerError, "create_click", "create click")
	}

	return nil
}

func generateShortCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
