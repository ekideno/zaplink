package handler_test

import (
	"context"

	"github.com/ekideno/zaplink/internal/model"
)

type mockLinkService struct {
	createFn         func(ctx context.Context, url string) (*model.Link, error)
	getByShortCodeFn func(ctx context.Context, shortCode string) (*model.Link, error)
}

func (m *mockLinkService) CreateLink(ctx context.Context, url string) (*model.Link, error) {
	if m.createFn != nil {
		return m.createFn(ctx, url)
	}
	return nil, nil
}

func (m *mockLinkService) GetByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	if m.getByShortCodeFn != nil {
		return m.getByShortCodeFn(ctx, shortCode)
	}
	return nil, nil
}
