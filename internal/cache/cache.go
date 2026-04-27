package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Enabled() bool {
	return c != nil && c.client != nil
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) bool {
	if !c.Enabled() {
		return false
	}
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(value), dest) == nil
}

func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if !c.Enabled() {
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, encoded, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}
	_ = c.client.Del(ctx, keys...).Err()
}

func (c *Cache) DeletePattern(ctx context.Context, pattern string) {
	if !c.Enabled() || pattern == "" {
		return
	}
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = c.client.Del(ctx, keys...).Err()
		}
		if next == 0 {
			return
		}
		cursor = next
	}
}

func (c *Cache) Client() *redis.Client {
	if !c.Enabled() {
		return nil
	}
	return c.client
}
