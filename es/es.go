package es

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	ErrNotFound         = errors.New("document not found")
	ErrInvalidQuery     = errors.New("invalid query")
	ErrRequestFailed    = errors.New("request failed")
	ErrDecodingFailed   = errors.New("response decoding failed")
	ErrIndexExists      = errors.New("index already exists")
	ErrIndexNotExist    = errors.New("index does not exist")
	ErrBulkOperation    = errors.New("bulk operation failed")
	ErrInvalidParameter = errors.New("invalid parameter")
)

type ESClient struct {
	*elasticsearch.Client
	config *Config
}

func NewESClient(cfg *Config) (*ESClient, error) {
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("%w: no addresses provided", ErrInvalidParameter)
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 20
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90
	}
	transport := &http.Transport{
		MaxIdleConnsPerHost:   cfg.MaxIdleConns,
		IdleConnTimeout:       time.Duration(cfg.IdleConnTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.Timeout) * time.Second,
	}
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		// MaxRetries: cfg.MaxRetries,
		Transport: transport,
	}

	// 初始化客户端
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	// 验证连接
	res, err := client.Ping()
	if err != nil {
		return nil, fmt.Errorf("%w: ping failed: %v", ErrRequestFailed, err)
	}

	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES ping error: %s", res.String())
	}

	return &ESClient{
		Client: client,
		config: cfg,
	}, nil
}

// IsNotFound 检查是否为文档不存在错误
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
