package redis

import (
	"github.com/redis/go-redis/v9"
)

// IsNil 检查是否为Redis空值错误
func IsNil(err error) bool {
	return err == redis.Nil
}
