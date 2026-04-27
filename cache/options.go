package cache

import "time"

type Options struct {
	DefaultTTL     time.Duration
	EnableCompress  bool
	CompressMinSize int
}
