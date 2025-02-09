package llm

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// ModelType defines supported model types
type ModelType string

const (
	ModelTypeOllama ModelType = "ollama"
	ModelTypeAliyun ModelType = "aliyun"
)

// LLMOptions 定义LLM配置选项
type ModelOptions struct {
	EnableEmbeddings bool
	HTTPTimeout      time.Duration
	MaxIdleConns     int
	// 其他通用配置...
}

// Config contains configuration for all supported models
type Config struct {
	Type   ModelType    `yaml:"type"`
	Ollama OllamaConfig `yaml:"ollama"`
	Aliyun AliyunConfig `yaml:"aliyun"`
}

// OllamaConfig contains configuration for Ollama LLM
type OllamaConfig struct {
	Address       string        `yaml:"address"`
	LLMModel      string        `yaml:"llm_model"`
	EmbedderModel string        `yaml:"embedder_model"`
	Temperature   float64       `yaml:"temperature"`
	Timeout       time.Duration `yaml:"timeout"`
	MaxIdleConns  int           `yaml:"max_idle_conns"`
}

type AliyunConfig struct {
	BaseURL        string        `yaml:"base_url"`
	APIKey         string        `yaml:"api_key"`
	LLMModel       string        `yaml:"llm_model"`
	EmbeddingModel string        `yaml:"embedding_model"`
	Timeout        time.Duration `yaml:"timeout"`
	MaxIdleConns   int           `yaml:"max_idle_conns"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("model type is required")
	}

	switch c.Type {
	case ModelTypeOllama:
		return c.Ollama.Validate()
	case ModelTypeAliyun:
		return c.Aliyun.Validate()
	default:
		return fmt.Errorf("unsupported model type: %s", c.Type)
	}
}

// Validate checks if Ollama configuration is valid
func (c *OllamaConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("ollama address is required")
	}
	if c.LLMModel == "" {
		return fmt.Errorf("ollama LLM model is required")
	}
	if c.EmbedderModel == "" {
		return fmt.Errorf("ollama embedder model is required")
	}
	return nil
}

// Validate checks if Aliyun configuration is valid
func (c *AliyunConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("aliyun base URL is required")
	}
	if c.GetAPIKey() == "" {
		return fmt.Errorf("aliyun API key is required")
	}
	if c.LLMModel == "" {
		return fmt.Errorf("aliyun LLM model is required")
	}
	if c.EmbeddingModel == "" {
		return fmt.Errorf("aliyun embedding model is required")
	}
	return nil
}

// GetAPIKey returns API key from environment variable if set, otherwise from config
func (c *AliyunConfig) GetAPIKey() string {
	if apiKey := os.Getenv("ALIYUN_API_KEY"); apiKey != "" {
		return apiKey
	}
	return c.APIKey
}

// createHTTPClient creates a configured HTTP client
func createHTTPClient(timeout time.Duration, maxIdleConns int) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConns,
			IdleConnTimeout:     timeout,
			DisableCompression:  true,
			MaxIdleConnsPerHost: maxIdleConns,
		},
	}
}
