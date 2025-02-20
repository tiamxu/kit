package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tiamxu/kit/log"
)

type RedisClient struct {
	*redis.Client
	config *Config
}

// NewClient new redis client
func NewClient(cfg *Config) (*RedisClient, error) {
	// 设置默认值
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 20
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 // 默认连接超时时间设为5秒
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 // 默认读写超时时间为10秒
	}
	option := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
	}

	client := redis.NewClient(option)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		Client: client,
		config: cfg,
	}, nil
}

// SetModelToCache save model to cache
func (r *RedisClient) SetModelToCache(key string, model interface{}, ttl time.Duration) error {
	ctx, cancel := r.getContextWithTimeout()
	defer cancel()

	log.Tracef("[cache: set_model_to_cache]: key=%s and ttl=%v", key, ttl)
	var (
		bs        []byte
		data      []byte
		err       error
		cacheFlag = CacheFormatJSON
		gziped    bool
	)

	if gziped, data, err = toGzipJSON(model); err != nil {
		return err
	}
	if gziped {
		cacheFlag = CacheFormatJSONGzip
	}
	if bs, err = json.Marshal(modelCacheItem{
		Data: data,
		Flag: uint32(cacheFlag),
	}); err != nil {
		return err
	}
	if _, err = r.Set(ctx, key, string(bs), ttl).Result(); err != nil {
		return err
	}
	return nil
}

// GetCacheToModel get cache to model

func (r *RedisClient) GetCacheToModel(key string, model interface{}) (bool, error) {
	ctx, cancel := r.getContextWithTimeout()
	defer cancel()

	log.Tracef("[cache: get_cache_to_model], key=%s", key)
	it, err := r.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	cacheItem := modelCacheItem{}
	if err = json.Unmarshal([]byte(it), &cacheItem); err != nil {
		log.Errorf("[cache:%s] Unmarshal value error, %s", key, err)
		return false, fmt.Errorf("unmarshal error for key %s: %w", key, err)
	}
	switch cacheItem.Flag {
	case CacheFormatJSON:
		err = json.Unmarshal(cacheItem.Data, model)
	case CacheFormatJSONGzip:
		err = fromGzipJSON(cacheItem.Data, model)
	default:
		err = fmt.Errorf("invalid cache formate %d", cacheItem.Flag)
	}
	if err != nil {
		log.Errorf("[cache:%s] %s", key, err)
		return false, err
	}
	return true, nil
}

func IsRedisNil(err error) bool {
	return err == redis.Nil
}

func IsNotNil(err error) bool {
	if err != nil && err != redis.Nil {
		return true
	}
	return false
}

// getContextWithTimeout 返回一个带有指定超时时间的上下文
func (r *RedisClient) getContextWithTimeout() (context.Context, context.CancelFunc) {
	timeout := time.Duration(r.config.Timeout) * time.Second
	return context.WithTimeout(context.Background(), timeout)
}
