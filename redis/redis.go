package pkgredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheEntry struct {
	Valid        bool              `json:"valid"`
	Subject      string            `json:"subject,omitempty"`
	Roles        []string          `json:"roles,omitempty"`
	Claims       map[string]string `json:"claims,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type Redis struct {
	rdb *redis.Client
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{rdb: rdb}
}

func (c *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (c *Redis) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Redis) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}
