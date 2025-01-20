package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type OllamaConfig struct {
	Address       string  `yaml:"address"`
	LLMModel      string  `yaml:"llm_model"`
	EmbedderModel string  `yaml:"embedder_model"`
	Temperature   float64 `yaml:"temperature"`
}

type ModelService struct {
	llm      llms.Model
	embedder embeddings.Embedder
}

func NewModelService() *ModelService {
	return &ModelService{}
}

func NewOllamaModels(cfg *OllamaConfig) (llms.Model, embeddings.Embedder, error) {
	llm, err := ollama.New(
		ollama.WithModel(cfg.LLMModel),
		ollama.WithServerURL(cfg.Address),
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
