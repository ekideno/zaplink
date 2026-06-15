// internal/service/mock_test.go
package service_test

import (
	"context"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/ekideno/zaplink/internal/repository"
)

type mockLinkRepo struct {
	createFn         func(ctx context.Context, link *model.Link) error
	getByIDFn        func(ctx context.Context, id int64) (*model.Link, error)
	getByShortCodeFn func(ctx context.Context, shortCode string) (*model.Link, error)
	updateFn         func(ctx context.Context, link *model.Link) error
}

type mockClickRepo struct {
	createFn func(ctx context.Context, click *model.Click) error
	countFn  func(ctx context.Context, linkID int64) (int64, error)
}

func (m *mockLinkRepo) CreateLink(ctx context.Context, link *model.Link) error {
	if m.createFn != nil {
		return m.createFn(ctx, link)
	}
	return nil
}

func (m *mockLinkRepo) GetLinkByID(ctx context.Context, id int64) (*model.Link, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *mockLinkRepo) GetLinkByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	if m.getByShortCodeFn != nil {
		return m.getByShortCodeFn(ctx, shortCode)
	}
	return nil, repository.ErrNotFound
}

func (m *mockLinkRepo) UpdateLink(ctx context.Context, link *model.Link) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, link)
	}
	return nil
}

func (m *mockClickRepo) CreateClick(ctx context.Context, click *model.Click) error {
	if m.createFn != nil {
		return m.createFn(ctx, click)
	}
	return nil
}

func (m *mockClickRepo) CountClicksByLinkID(ctx context.Context, linkID int64) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, linkID)
	}
	return 0, nil
}
