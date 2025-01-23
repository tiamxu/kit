package llm

import (
	"fmt"

	"github.com/tiamxu/kit/log"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// NewModels initializes the appropriate LLM based on config
func NewModels(cfg *Config) (llms.Model, embeddings.Embedder, error) {
	if err := cfg.Validate(); err != nil {
		log.Printf("config validation failed: %v", err)
		return nil, nil, fmt.Errorf("config validation failed: %w", err)
	}

	switch cfg.Type {
	case ModelTypeOllama:
		return NewOllamaModels(&cfg.Ollama)
	case ModelTypeAliyun:
		return NewAliyunModels(&cfg.Aliyun)
	default:
		return nil, nil, fmt.Errorf("unsupported LLM type: %s", cfg.Type)
	}
}

// NewOllamaModels initializes Ollama LLM and embedder
func NewOllamaModels(cfg *OllamaConfig) (llms.Model, embeddings.Embedder, error) {
	// if err := cfg.Validate(); err != nil {
	// 	log.Printf("Ollama config validation failed: %v", err)
	// 	return nil, nil, fmt.Errorf("ollama config validation failed: %w", err)
	// }

	httpClient := createHTTPClient(cfg.Timeout, cfg.MaxIdleConns)
	llm, err := ollama.New(
		ollama.WithModel(cfg.LLMModel),
		ollama.WithServerURL(cfg.Address),
		ollama.WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Printf("Failed to initialize LLM: %v", err)
		return nil, nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	embedderModel, err := ollama.New(
		ollama.WithModel(cfg.EmbedderModel),
		ollama.WithServerURL(cfg.Address),
	)
	if err != nil {
		log.Printf("Failed to initialize embedder model: %v", err)
		return nil, nil, fmt.Errorf("failed to initialize embedder model: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(embedderModel)
	if err != nil {
		log.Printf("Failed to create embedder: %v", err)
		return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return llm, embedder, nil
}

func NewAliyunModels(cfg *AliyunConfig) (llms.Model, embeddings.Embedder, error) {
	// if err := cfg.Validate(); err != nil {
	// 	log.Printf("Aliyun config validation failed: %v", err)
	// 	return nil, nil, fmt.Errorf("aliyun config validation failed: %w", err)
	// }

	httpClient := createHTTPClient(cfg.Timeout, cfg.MaxIdleConns)

	llm, err := openai.New(
		openai.WithBaseURL(cfg.BaseURL),
		openai.WithToken(cfg.GetAPIKey()),
		openai.WithModel(cfg.LLMModel),
		openai.WithEmbeddingModel(cfg.EmbeddingModel),
		openai.WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Printf("Failed to initialize Aliyun LLM: %v", err)
		return nil, nil, fmt.Errorf("failed to initialize Aliyun LLM: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Printf("Failed to create embedder: %v", err)
		return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return llm, embedder, nil
}
