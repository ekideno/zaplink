package model

import "time"

type Link struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	IsActive    bool
	CreatedAt   time.Time
}
