package cache

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

// Cache 缓存接口定义
type Cache interface {
	// Get 获取字符串值，未命中返回 ErrNotFound
	Get(ctx context.Context, key string) (string, error)

	// Set 设置字符串值
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Delete 删除一个或多个key
	Delete(ctx context.Context, keys ...string) error

	// Exists 批量检查key是否存在
	Exists(ctx context.Context, keys ...string) (map[string]bool, error)

	// MGet 批量获取字符串值，不存在的key返回nil
	MGet(ctx context.Context, keys ...string) (map[string]*string, error)

	// MSet 批量设置字符串值
	MSet(ctx context.Context, kv map[string]string, ttl time.Duration) error

	// GetObj 获取对象（反序列化 + 解压），未命中返回 false, nil
	GetObj(ctx context.Context, key string, dest interface{}) (bool, error)

	// SetObj 设置对象（序列化 + 压缩）
	SetObj(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// GetTTL 获取key的剩余过期时间
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Expire 设置key的过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Incr 原子自增
	Incr(ctx context.Context, key string) (int64, error)

	// IncrBy 原子增加指定值
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// TryLock 尝试获取分布式锁，返回是否获取成功
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock 释放分布式锁
	Unlock(ctx context.Context, key string) error

	// Close 关闭缓存连接
	Close() error
}
