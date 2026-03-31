package cache

import "time"

// Options 缓存配置
type Options struct {
	// DefaultTTL 默认过期时间，0 表示不过期
	DefaultTTL time.Duration
	// NilCacheTTL 空值缓存TTL（防缓存穿透），0 表示不缓存空值
	NilCacheTTL time.Duration
	// EnableCompress 是否启用 Gzip 压缩
	EnableCompress bool
	// CompressMinSize 压缩阈值（字节），小于此值不压缩
	CompressMinSize int
}
