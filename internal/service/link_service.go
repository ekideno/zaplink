package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

type LinkService struct {
	repo repository.LinkRepository
}

func NewLinkService(repo repository.LinkRepository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) CreateLink(ctx context.Context, originalURL string) (*model.Link, error) {
	u, err := url.ParseRequestURI(originalURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("%w: must be http or https", ErrInvalidURL)
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

		if err := s.repo.CreateLink(ctx, link); err != nil {
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
	link, err := s.repo.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get link by short code: %w", err)
	}

	if !link.IsActive {
		return nil, ErrInactiveLink
	}

	return link, nil
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
