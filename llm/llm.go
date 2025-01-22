package llm

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// LLMType defines supported LLM types
type LLMType string

const (
	Ollama LLMType = "ollama"
	Aliyun LLMType = "aliyun"
)

// Config contains configuration for all supported LLMs
type Config struct {
	ModelType LLMType      `yaml:"model_type"`
	Ollama    OllamaConfig `yaml:"ollama"`
	Aliyun    AliyunConfig `yaml:"aliyun"`
}

// OllamaConfig contains configuration for Ollama LLM
type OllamaConfig struct {
	Address       string  `yaml:"address"`
	LLMModel      string  `yaml:"llm_model"`
	EmbedderModel string  `yaml:"embedder_model"`
	Temperature   float64 `yaml:"temperature"`
}

type AliyunConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	LLMModel       string `yaml:"llm_model"`
	EmbeddingModel string `yaml:"embedding_model"`
}

// GetAPIKey returns API key from environment variable if set, otherwise from config
func (c *AliyunConfig) GetAPIKey() string {
	if apiKey := os.Getenv("ALIYUN_API_KEY"); apiKey != "" {
		return apiKey
	}
	return c.APIKey
}

// ModelService provides unified interface for all LLMs
// type ModelService struct {
// 	llm      llms.Model
// 	embedder embeddings.Embedder
// }

// func NewModelService() *ModelService {
// 	return &ModelService{}
// }

// NewModels initializes the appropriate LLM based on config
func NewModels(cfg *Config) (llms.Model, embeddings.Embedder, error) {
	switch cfg.ModelType {
	case Ollama:
		return NewOllamaModels(&cfg.Ollama)
	case Aliyun:
		return NewAliyunModels(&cfg.Aliyun)
	default:
		return nil, nil, fmt.Errorf("unsupported LLM type: %s", cfg.ModelType)
	}
}

// NewOllamaModels initializes Ollama LLM and embedder
func NewOllamaModels(cfg *OllamaConfig) (llms.Model, embeddings.Embedder, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
			MaxIdleConnsPerHost: 10,
		},
	}
	llm, err := ollama.New(
		ollama.WithModel(cfg.LLMModel),
		ollama.WithServerURL(cfg.Address),
		ollama.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	embedderModel, err := ollama.New(
		ollama.WithModel(cfg.EmbedderModel),
		ollama.WithServerURL(cfg.Address),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize embedder model: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(embedderModel)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return llm, embedder, nil
}

func NewAliyunModels(cfg *AliyunConfig) (llms.Model, embeddings.Embedder, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
			MaxIdleConnsPerHost: 10,
		},
	}

	llm, err := openai.New(
		openai.WithBaseURL(cfg.BaseURL),
		openai.WithToken(cfg.GetAPIKey()),
		openai.WithModel(cfg.LLMModel),
		openai.WithEmbeddingModel(cfg.EmbeddingModel),
		openai.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化失败 Aliyun LLM: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, nil, fmt.Errorf("创建失败 embedder: %w", err)
	}

	return llm, embedder, nil
}
