package repository

import (
	"context"
	"errors"

	"github.com/ekideno/zaplink/internal/model"
)

var ErrNotFound = errors.New("not found")

type LinkRepository interface {
	CreateLink(ctx context.Context, link *model.Link) error
	GetLinkByID(ctx context.Context, id int64) (*model.Link, error)
	GetLinkByShortCode(ctx context.Context, shortCode string) (*model.Link, error)
	UpdateLink(ctx context.Context, link *model.Link) error
}

type ClickRepository interface {
	CreateClick(ctx context.Context, click *model.Click) error
}
