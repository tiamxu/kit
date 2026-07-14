package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	kiterrors "github.com/tiamxu/kit/errors"
)

const (
	defaultPoolSize     = 20
	defaultMaxIdle      = 15
	defaultDialTimeout  = 5
	defaultReadTimeout  = 10
	defaultWriteTimeout = 10
)

const (
	lockScript = `
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

type Config struct {
	Address      string `yaml:"address" json:"address"`
	Password     string `yaml:"password" json:"-"`
	DB           int    `yaml:"db" json:"db"`
	PoolSize     int    `yaml:"pool_size" json:"pool_size"`
	MaxIdle      int    `yaml:"max_idle" json:"max_idle"`
	MinIdle      int    `yaml:"min_idle" json:"min_idle"`
	DialTimeout  int    `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  int    `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout" json:"write_timeout"`
}

type Client struct {
	*redis.Client
}

func NewClient(cfg *Config) (*Client, error) {
	return NewClientCtx(context.Background(), cfg)
}

func NewClientCtx(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, kiterrors.Wrap("REDIS_PARAM", "config cannot be nil", kiterrors.ErrInvalidParam)
	}
	if ctx == nil {
		return nil, kiterrors.Wrap("REDIS_PARAM", "context cannot be nil", kiterrors.ErrInvalidParam)
	}
	cfgCopy := *cfg
	if cfgCopy.PoolSize <= 0 {
		cfgCopy.PoolSize = defaultPoolSize
	}
	if cfgCopy.MaxIdle <= 0 {
		cfgCopy.MaxIdle = defaultMaxIdle
	}
	if cfgCopy.DialTimeout <= 0 {
		cfgCopy.DialTimeout = defaultDialTimeout
	}
	if cfgCopy.ReadTimeout <= 0 {
		cfgCopy.ReadTimeout = defaultReadTimeout
	}
	if cfgCopy.WriteTimeout <= 0 {
		cfgCopy.WriteTimeout = defaultWriteTimeout
	}

	options := &redis.Options{
		Addr:         cfgCopy.Address,
		Password:     cfgCopy.Password,
		DB:           cfgCopy.DB,
		PoolSize:     cfgCopy.PoolSize,
		MinIdleConns: cfgCopy.MinIdle,
		MaxIdleConns: cfgCopy.MaxIdle,
		DialTimeout:  time.Duration(cfgCopy.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfgCopy.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfgCopy.WriteTimeout) * time.Second,
	}

	client := redis.NewClient(options)

	pingCtx, cancel := context.WithTimeout(ctx, time.Duration(cfgCopy.DialTimeout)*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, kiterrors.Wrap("REDIS_CONNECT", "failed to connect to Redis", err)
	}

	return &Client{Client: client}, nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}

var CacheErrNotFound = kiterrors.CacheErrNotFound

func IsNotFound(err error) bool {
	return kiterrors.IsNotFound(err)
}

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
	result, err := l.client.Eval(ctx, lockScript, []string{l.key}, l.token).Result()
	if err != nil {
		return err
	}
	if result != int64(1) {
		return kiterrors.Wrap("REDIS_UNLOCK", "lock token mismatch or lock not found", kiterrors.CacheErrLockFail)
	}
	return nil
}

func (l *RedisLock) Refresh(ctx context.Context, ttl time.Duration) error {
	result, err := l.client.Eval(ctx, unlockRefreshScript, []string{l.key}, l.token, ttl.Milliseconds()).Result()
	if err != nil {
		return err
	}
	if result != int64(1) {
		return kiterrors.Wrap("REDIS_LOCK_REFRESH", "lock token mismatch or lock not found", kiterrors.CacheErrLockFail)
	}
	return nil
}

type lockConfig struct {
	retryCount int
	retryDelay time.Duration
}

type LockOption func(*lockConfig)

func WithLockRetry(count int, delay time.Duration) LockOption {
	return func(c *lockConfig) {
		c.retryCount = count
		c.retryDelay = delay
	}
}

var defaultLockRetryCount = 3
var defaultLockRetryDelay = 200 * time.Millisecond

func (c *Client) TryLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (Lock, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	token := uuid.New().String()

	cfg := lockConfig{
		retryCount: defaultLockRetryCount,
		retryDelay: defaultLockRetryDelay,
	}
	for _, o := range opts {
		o(&cfg)
	}

	for i := 0; i < cfg.retryCount; i++ {
		ok, err := c.Client.SetNX(ctx, lockKey, token, ttl).Result()
		if err != nil {
			return nil, err
		}
		if ok {
			return &RedisLock{client: c.Client, key: lockKey, token: token}, nil
		}
		if i < cfg.retryCount-1 {
			select {
			case <-ctx.Done():
				return nil, kiterrors.Wrap("REDIS_LOCK", "context cancelled", ctx.Err())
			case <-time.After(cfg.retryDelay):
			}
		}
	}
	return nil, kiterrors.Wrap("REDIS_LOCK", "lock failed after retries", kiterrors.CacheErrLockFail)
}

func serialize(obj any) ([]byte, error) {
	bs, err := json.Marshal(obj)
	if err != nil {
		return nil, kiterrors.Wrap("REDIS_SERIALIZE", "marshal failed", err)
	}
	return bs, nil
}

func deserialize(data []byte, dest any) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return kiterrors.Wrap("REDIS_DESERIALIZE", "unmarshal failed", err)
	}
	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", CacheErrNotFound
		}
		return "", err
	}
	return val, nil
}

func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.Client.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.Client.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, keys ...string) (map[string]bool, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	pipe := c.Client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.Exists(ctx, k)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, kiterrors.Wrap("REDIS_EXISTS", "failed to check key existence", err)
	}
	exists := make(map[string]bool, len(keys))
	for i, cmd := range cmds {
		exists[keys[i]] = cmd.Val() > 0
	}
	return exists, nil
}

func (c *Client) MGet(ctx context.Context, keys ...string) (map[string]*string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	result, err := c.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	values := make(map[string]*string, len(keys))
	for i, v := range result {
		if v == nil {
			values[keys[i]] = nil
		} else {
			s, ok := v.(string)
			if !ok {
				return nil, kiterrors.Wrap("REDIS_OPERATION", "expected string value", nil)
			}
			values[keys[i]] = &s
		}
	}
	return values, nil
}

func (c *Client) MSet(ctx context.Context, kv map[string]string, ttl time.Duration) error {
	if len(kv) == 0 {
		return nil
	}
	pipe := c.Client.Pipeline()
	for k, v := range kv {
		pipe.Set(ctx, k, v, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) GetObj(ctx context.Context, key string, dest any) (bool, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	if err := deserialize([]byte(val), dest); err != nil {
		return false, kiterrors.Wrap("REDIS_DESERIALIZE", "deserialize failed", err)
	}
	return true, nil
}

func (c *Client) SetObj(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := serialize(value)
	if err != nil {
		return err
	}
	return c.Client.Set(ctx, key, data, ttl).Err()
}

func (c *Client) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.Client.TTL(ctx, key).Result()
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.Client.Expire(ctx, key, ttl).Err()
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.Client.Incr(ctx, key).Result()
}

func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.Client.IncrBy(ctx, key, value).Result()
}

func (c *Client) Scan(ctx context.Context, match string, count int64) ([]string, error) {
	keys := make([]string, 0, count)
	var cursor uint64
	for {
		var batch []string
		var err error
		if match != "" {
			batch, cursor, err = c.Client.Scan(ctx, cursor, match, count).Result()
		} else {
			batch, cursor, err = c.Client.Scan(ctx, cursor, "*", count).Result()
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}
