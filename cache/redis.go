package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tiamxu/kit/redis"
)

const (
	defaultCompressMinSize = 2048
	lockScript             = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`
	unlockRefreshScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
else
    return 0
end
`
)

type Lock interface {
	Unlock(ctx context.Context) error
	Refresh(ctx context.Context, ttl time.Duration) error
}

type RedisLock struct {
	client *redis.Client
	key    string
	token  string
}

func (l *RedisLock) Unlock(ctx context.Context) error {
	_, err := l.client.Eval(ctx, lockScript, []string{l.key}, l.token).Result()
	return err
}

func (l *RedisLock) Refresh(ctx context.Context, ttl time.Duration) error {
	_, err := l.client.Eval(ctx, unlockRefreshScript, []string{l.key}, l.token, ttl.Milliseconds()).Result()
	return err
}

type lockConfig struct {
	retryCount int
	retryDelay time.Duration
}

type LockOption func(*lockConfig)

func WithRetry(count int, delay time.Duration) LockOption {
	return func(c *lockConfig) {
		c.retryCount = count
		c.retryDelay = delay
	}
}

var defaultLockRetryCount = 3
var defaultLockRetryDelay = 200 * time.Millisecond

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
	exists := make(map[string]bool, len(keys))
	for _, k := range keys {
		n, err := c.client.Exists(ctx, k).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check key existence: %w", err)
		}
		exists[k] = n > 0
	}
	return exists, nil
}

// MGet 批量获取字符串值，不存在的key返回nil
func (c *RedisCache) MGet(ctx context.Context, keys ...string) (map[string]*string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	values, err := c.client.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*string, len(keys))
	for i, v := range values {
		if v == nil {
			result[keys[i]] = nil
		} else {
			result[keys[i]] = v
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

// TryLock 尝试获取分布式锁，支持重试
func (c *RedisCache) TryLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (Lock, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	token := uuid.New().String()

	cfg := lockConfig{
		retryCount: defaultLockRetryCount,
		retryDelay: defaultLockRetryDelay,
	}
	for _, o := range opts {
		o(&cfg)
	}

	var err error
	for i := 0; i < cfg.retryCount; i++ {
		ok, err := c.client.SetNX(ctx, lockKey, token, ttl).Result()
		if err != nil {
			return nil, err
		}
		if ok {
			return &RedisLock{
				client: c.client,
				key:    lockKey,
				token:  token,
			}, nil
		}
		if i < cfg.retryCount-1 {
			time.Sleep(cfg.retryDelay)
		}
	}
	return nil, fmt.Errorf("lock failed after %d retries: %w", cfg.retryCount, err)
}

// Close 关闭缓存连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}