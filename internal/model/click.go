package model

import "time"

type Click struct {
	ID        int64
	LinkID    int64
	CreatedAt time.Time
	UserAgent *string
	IPAddress *string
}
