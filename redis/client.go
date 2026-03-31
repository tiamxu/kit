package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultPoolSize    = 20
	defaultMaxIdle     = 15
	defaultDialTimeout = 5
	defaultReadTimeout = 10
	defaultWriteTimeout = 10
)

// Client Redis客户端封装
type Client struct {
	*redis.Client
}

// NewClient 创建Redis客户端
func NewClient(cfg *Config) (*Client, error) {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = defaultPoolSize
	}
	if cfg.MaxIdle <= 0 {
		cfg.MaxIdle = defaultMaxIdle
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = defaultDialTimeout
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}

	options := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdle,
		MaxIdleConns: cfg.MaxIdle,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	}

	client := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.DialTimeout)*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// MGet 批量获取多个key的值，不存在的key返回nil
func (c *Client) MGet(ctx context.Context, keys ...string) ([]*string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	result, err := c.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	values := make([]*string, len(result))
	for i, v := range result {
		if v == nil {
			values[i] = nil
		} else {
			s := v.(string)
			values[i] = &s
		}
	}
	return values, nil
}

// MSet 批量设置多个key-value
func (c *Client) MSet(ctx context.Context, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	pairs := make([]interface{}, 0, len(kv)*2)
	for k, v := range kv {
		pairs = append(pairs, k, v)
	}
	return c.Client.MSet(ctx, pairs...).Err()
}

// Scan 遍历匹配的key，替代KEYS命令
func (c *Client) Scan(ctx context.Context, match string, count int64) ([]string, error) {
	var keys []string
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
