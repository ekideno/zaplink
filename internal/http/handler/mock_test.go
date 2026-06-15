package handler_test

import (
	"context"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/service"
)

type mockLinkService struct {
	createFn         func(ctx context.Context, url string) (*model.Link, error)
	getByShortCodeFn func(ctx context.Context, shortCode string) (*model.Link, error)
	getLinkInfoFn    func(ctx context.Context, shortCode string) (*service.LinkInfo, error)
	recordClickFn    func(ctx context.Context, click *model.Click) error
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

func (m *mockLinkService) GetLinkInfo(ctx context.Context, shortCode string) (*service.LinkInfo, error) {
	if m.getLinkInfoFn != nil {
		return m.getLinkInfoFn(ctx, shortCode)
	}
	return nil, nil
}

func (m *mockLinkService) TrackClick(ctx context.Context, linkID int64, userAgent, remoteAddr string) error {
	if m.recordClickFn != nil {
		return m.recordClickFn(ctx, &model.Click{
			LinkID:    linkID,
			UserAgent: &userAgent,
			IPAddress: &remoteAddr,
		})
	}
	return nil
}
