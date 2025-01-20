package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// LLMType defines supported LLM types
type LLMType string

const (
	Ollama   LLMType = "ollama"
	DeepSeek LLMType = "deepseek"
)

// Config contains configuration for all supported LLMs
type Config struct {
	Type          LLMType `yaml:"type"`
	OllamaConfig  `yaml:",inline"`
	DeepSeekConfig `yaml:",inline"`
}

// OllamaConfig contains configuration for Ollama LLM
type OllamaConfig struct {
	Address       string  `yaml:"address"`
	LLMModel      string  `yaml:"llm_model"`
	EmbedderModel string  `yaml:"embedder_model"`
	Temperature   float64 `yaml:"temperature"`
}

// DeepSeekConfig contains configuration for DeepSeek LLM
type DeepSeekConfig struct {
	APIKey        string  `yaml:"api_key"`
	Model         string  `yaml:"model"`
	BaseURL       string  `yaml:"base_url"`
	Timeout       int     `yaml:"timeout"`
	Temperature   float64 `yaml:"temperature"`
	MaxTokens     int     `yaml:"max_tokens"`
}

// ModelService provides unified interface for all LLMs
type ModelService struct {
	llm      llms.Model
	embedder embeddings.Embedder
}

func NewModelService() *ModelService {
	return &ModelService{}
}

func (ms *ModelService) Generate(ctx context.Context, prompt string) (string, error) {
	result, err := ms.llm.GenerateContent(ctx, []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	}, llms.WithTemperature(0.7), llms.WithMaxTokens(2048))
	if err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Content, nil
}

// NewModels initializes the appropriate LLM based on config
func NewModels(cfg *Config) (llms.Model, embeddings.Embedder, error) {
	switch cfg.Type {
	case Ollama:
		return NewOllamaModels(&cfg.OllamaConfig)
	case DeepSeek:
		return NewDeepSeekModels(&cfg.DeepSeekConfig)
	default:
		return nil, nil, fmt.Errorf("unsupported LLM type: %s", cfg.Type)
	}
}

// NewOllamaModels initializes Ollama LLM and embedder
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

// NewDeepSeekModels initializes DeepSeek LLM and embedder
func NewDeepSeekModels(cfg *DeepSeekConfig) (llms.Model, embeddings.Embedder, error) {
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	llm := &DeepSeekLLM{
		client:  client,
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DeepSeek embedder: %w", err)
	}

	return llm, embedder, nil
}

// DeepSeekLLM implements the DeepSeek API
type DeepSeekLLM struct {
	client    *http.Client
	apiKey    string
	baseURL   string
	model     string
	timeout   time.Duration
}

func (ds *DeepSeekLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	result, err := ds.GenerateContent(ctx, []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	}, options...)
	if err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Content, nil
}

func (ds *DeepSeekLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	reqBody := map[string]interface{}{
		"model":       ds.model,
		"messages":    messages,
		"temperature": 0.7,
		"max_tokens":  2048,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ds.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ds.apiKey)

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error.Message != "" {
		return nil, fmt.Errorf("api error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response.Choices[0].Message.Content,
			},
		},
	}, nil
}

func (ds *DeepSeekLLM) CreateEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": ds.model,
		"input": texts,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ds.baseURL+"/v1/embeddings", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ds.apiKey)

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error.Message != "" {
		return nil, fmt.Errorf("api error: %s", response.Error.Message)
	}

	embeddings := make([][]float32, len(response.Data))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}
