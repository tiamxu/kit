package cache

import (
	"context"
	"time"

	"github.com/tiamxu/kit/redis"
)

func NewRedisCache(c *redis.RedisClient) *RedisCache {
	return &RedisCache{client: c}
}

type RedisCache struct {
	client *redis.RedisClient
}

func (c *RedisCache) GetClient() *redis.RedisClient {
	return c.client
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	str, err := c.client.Get(ctx, key).Result()
	if redis.IsRedisNil(err) {
		return "", nil
	}
	return str, err
}

func (c *RedisCache) GetObj(ctx context.Context, key string, model interface{}) (bool, error) {
	return c.client.GetCacheToModel(ctx, key, model)
}

func (c *RedisCache) Set(ctx context.Context, key, value string, ttlSecond int32) error {
	return c.client.Set(ctx, key, value, time.Second*time.Duration(ttlSecond)).Err()
}

func (c *RedisCache) SetObj(ctx context.Context, key string, obj interface{}, ttlSecond int32) error {
	return c.client.SetModelToCache(ctx, key, obj, time.Second*time.Duration(ttlSecond))
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
