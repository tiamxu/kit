package redis

import (
	"bytes"
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
		cfg.DialTimeout = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	if cfg.MaxIdle <= 0 {
		cfg.MaxIdle = 15 // 默认最大空闲连接
	}
	if cfg.MinIdle < 0 {
		cfg.MinIdle = 0 // 确保最小空闲连接不为负数
	}
	// 设置压缩阈值默认值
	if cfg.GzipMinSize <= 0 {
		cfg.GzipMinSize = 2048 // 提高默认阈值
	}
	option := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdle,
		MaxIdleConns: cfg.MaxIdle,
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
	log.Tracef("[cache: set_model_to_cache]: key=%s and ttl=%v", key, ttl)

	ctx, cancel := r.getContextWithTimeout()
	defer cancel()

	// 使用优化后的压缩方法
	gziped, data, err := toGzipJSON(model, r.config.GzipMinSize)
	if err != nil {
		return fmt.Errorf("SetModelToCache[compress] key=%s: %w", key, err)
	}

	// 根据压缩情况设置标志位
	var flag uint32
	if gziped {
		flag = CacheFormatJSONGzip
	} else {
		flag = CacheFormatJSON
	}

	cacheItem := modelCacheItem{
		Data: data,
		Flag: flag,
	}

	// 复用内存池
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	enc := json.NewEncoder(buf)
	if err := enc.Encode(cacheItem); err != nil {
		return fmt.Errorf("SetModelToCache[encode] key=%s: %w", key, err)
	}

	var lastErr error
	for i := 0; i <= r.config.RetryTimes; i++ {
		if _, err = r.Set(ctx, key, buf.Bytes(), ttl).Result(); err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(i*100) * time.Millisecond) // 线性退避
	}
	return fmt.Errorf("SetModelToCache failed after %d retries: %w", r.config.RetryTimes, lastErr)
}

// GetCacheToModel get cache to model

func (r *RedisClient) GetCacheToModel(key string, model interface{}) (bool, error) {
	ctx, cancel := r.getContextWithTimeout()
	defer cancel()

	// 带重试的读取
	var (
		value string
		err   error
	)
	for i := 0; i <= r.config.RetryTimes; i++ {
		if value, err = r.Get(ctx, key).Result(); err == nil {
			break
		}
		if err == redis.Nil {
			return false, nil
		}
		time.Sleep(time.Duration(i*100) * time.Millisecond)
	}
	if err != nil {
		return false, fmt.Errorf("GetCacheToModel[key=%s]: %w", key, err)
	}

	// 优化解码流程
	var cacheItem modelCacheItem
	dec := json.NewDecoder(bytes.NewReader([]byte(value)))
	dec.UseNumber() // 防止数值类型失真
	if err := dec.Decode(&cacheItem); err != nil {
		return false, fmt.Errorf("GetCacheToModel[decode_header] key=%s: %w", key, err)
	}

	// 优化解压分支判断
	switch cacheItem.Flag {
	case CacheFormatJSON:
		if err := json.Unmarshal(cacheItem.Data, model); err != nil {
			return false, fmt.Errorf("GetCacheToModel[unmarshal] key=%s: %w", key, err)
		}
	case CacheFormatJSONGzip:
		if err := fromGzipJSON(cacheItem.Data, model); err != nil {
			return false, fmt.Errorf("GetCacheToModel[unzip] key=%s: %w", key, err)
		}
	default:
		return false, fmt.Errorf("GetCacheToModel[invalid_flag] key=%s flag=%d", key, cacheItem.Flag)
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
