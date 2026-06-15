package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: "link:",
	}
}

func (c *RedisCache) GetByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	data, err := c.client.Get(ctx, c.key(shortCode)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("get link from cache: %w", err)
	}

	var link model.Link
	if err := json.Unmarshal(data, &link); err != nil {
		return nil, fmt.Errorf("unmarshal cached link: %w", err)
	}

	return &link, nil
}

func (c *RedisCache) SetLink(ctx context.Context, link *model.Link, ttl time.Duration) error {
	data, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("marshal link: %w", err)
	}

	if err := c.client.Set(ctx, c.key(link.ShortCode), data, ttl).Err(); err != nil {
		return fmt.Errorf("set link in cache: %w", err)
	}

	return nil
}

func (c *RedisCache) DeleteByShortCode(ctx context.Context, shortCode string) error {
	if err := c.client.Del(ctx, c.key(shortCode)).Err(); err != nil {
		return fmt.Errorf("delete link from cache: %w", err)
	}

	return nil
}

func (c *RedisCache) key(shortCode string) string {
	return c.prefix + shortCode
}
