package errors

import (
	"errors"
	"fmt"

	"github.com/tiamxu/kit/log"
)

// 基础错误类型
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Sentinel Errors - 通用
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidParam = errors.New("invalid parameter")
	ErrTimeout      = errors.New("operation timeout")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal error")
	ErrConnect      = errors.New("connection error")
)

// Sentinel Errors - cache 模块
var (
	CacheErrNotFound    = errors.New("cache: key not found")
	CacheErrLockFail    = errors.New("cache: lock failed")
	CacheErrSerialize   = errors.New("cache: serialize error")
	CacheErrDeserialize = errors.New("cache: deserialize error")
)

// Sentinel Errors - sql 模块
var (
	SqlErrNoRows      = errors.New("sql: no rows")
	SqlErrConnect     = errors.New("sql: connection error")
	SqlErrQuery       = errors.New("sql: query error")
	SqlErrTransaction = errors.New("sql: transaction error")
)

// Sentinel Errors - redis 模块
var (
	RedisErrConnect   = errors.New("redis: connection error")
	RedisErrOperation = errors.New("redis: operation error")
	RedisErrNil       = errors.New("redis: nil value")
)

// Sentinel Errors - kafka 模块
var (
	KafkaErrProduce   = errors.New("kafka: produce error")
	KafkaErrConsume   = errors.New("kafka: consume error")
	KafkaErrNoBrokers = errors.New("kafka: no brokers")
	KafkaErrTopic     = errors.New("kafka: topic error")
)

// IsNotFound 统一判断"未找到"错误
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, CacheErrNotFound) ||
		errors.Is(err, SqlErrNoRows)
}

// IsTimeout 统一判断"超时"错误
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, RedisErrOperation)
}

// IsConnect 统一判断"连接"错误
func IsConnect(err error) bool {
	if errors.Is(err, ErrConnect) ||
		errors.Is(err, SqlErrConnect) ||
		errors.Is(err, RedisErrConnect) {
		return true
	}
	switch GetCode(err) {
	case "SQL_CONNECT", "SQL_PING", "REDIS_CONNECT":
		return true
	default:
		return false
	}
}

// Wrap 错误包装（带错误码），不打印日志
func Wrap(code, msg string, cause error) error {
	return &Error{Code: code, Message: msg, Cause: cause}
}

// LogError 打印日志并返回错误（用于记录错误）
func LogError(code, msg string, cause error) error {
	if cause != nil {
		log.Errorf("%s: %s: %v", code, msg, cause)
	}
	return &Error{Code: code, Message: msg, Cause: cause}
}

// GetCode 获取错误码
func GetCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
