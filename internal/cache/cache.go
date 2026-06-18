package cache

import (
	"context"
	"time"

	"github.com/ekideno/zaplink/internal/model"
)

type LinkCache interface {
	GetByShortCode(ctx context.Context, shortCode string) (*model.Link, error)
	SetLink(ctx context.Context, link *model.Link, ttl time.Duration) error
	DeleteByShortCode(ctx context.Context, shortCode string) error
	Ping(ctx context.Context) error
}
