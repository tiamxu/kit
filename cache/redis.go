package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/tiamxu/kit/redis"
)

const (
	defaultCompressMinSize = 2048
	lockScript            = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`
)

// RedisCache 基于Redis的缓存实现
type RedisCache struct {
	client *redis.Client
	opts   Options
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(client *redis.Client, opts Options) *RedisCache {
	if opts.CompressMinSize <= 0 {
		opts.CompressMinSize = defaultCompressMinSize
	}
	return &RedisCache{
		client: client,
		opts:   opts,
	}
}

// Get 获取字符串值，未命中返回 ErrNotFound
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if redis.IsNil(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return val, nil
}

// Set 设置字符串值
func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.opts.DefaultTTL
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete 删除一个或多个key
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// Exists 批量检查key是否存在
func (c *RedisCache) Exists(ctx context.Context, keys ...string) (map[string]bool, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	result, err := c.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	exists := make(map[string]bool, len(keys))
	for _, k := range keys {
		exists[k] = false
	}
	if result > 0 {
		for _, k := range keys {
			n, _ := c.client.Exists(ctx, k).Result()
			exists[k] = n > 0
		}
	}
	return exists, nil
}

// MGet 批量获取字符串值，不存在的key返回nil
func (c *RedisCache) MGet(ctx context.Context, keys ...string) (map[string]*string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	values, err := c.client.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]*string, len(keys))
	for i, v := range values {
		if v == nil {
			result[keys[i]] = nil
		} else {
			s := v.(string)
			result[keys[i]] = &s
		}
	}
	return result, nil
}

// MSet 批量设置字符串值
func (c *RedisCache) MSet(ctx context.Context, kv map[string]string, ttl time.Duration) error {
	if len(kv) == 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = c.opts.DefaultTTL
	}
	pipe := c.client.Pipeline()
	for k, v := range kv {
		pipe.Set(ctx, k, v, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// GetObj 获取对象（反序列化 + 解压），未命中返回 false, nil
func (c *RedisCache) GetObj(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if redis.IsNil(err) {
			return false, nil
		}
		return false, err
	}
	if err := deserialize([]byte(val), dest); err != nil {
		return false, fmt.Errorf("cache: deserialize key=%s: %w", key, err)
	}
	return true, nil
}

// SetObj 设置对象（序列化 + 压缩）
func (c *RedisCache) SetObj(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.opts.DefaultTTL
	}
	data, err := serialize(value, c.opts.EnableCompress, c.opts.CompressMinSize)
	if err != nil {
		return fmt.Errorf("cache: serialize key=%s: %w", key, err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetTTL 获取key的剩余过期时间
func (c *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Expire 设置key的过期时间
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// Incr 原子自增
func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// IncrBy 原子增加指定值
func (c *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// TryLock 尝试获取分布式锁
// 返回 true 表示获取成功，false 表示锁已被占用
func (c *RedisCache) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := c.client.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return true, nil
}

// Unlock 释放分布式锁（使用 Lua 脚本保证原子性）
func (c *RedisCache) Unlock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := c.client.Eval(ctx, lockScript, []string{lockKey}, token).Result()
	return err
}

// Close 关闭缓存连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}
