package cache

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (map[string]bool, error)
	MGet(ctx context.Context, keys ...string) (map[string]*string, error)
	MSet(ctx context.Context, kv map[string]string, ttl time.Duration) error
	GetObj(ctx context.Context, key string, dest interface{}) (bool, error)
	SetObj(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	TryLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (Lock, error)
	Close() error
}