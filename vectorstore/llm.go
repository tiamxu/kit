package vector

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/milvus"
)

type ModelService struct {
	llm      llms.Model
	embedder embeddings.Embedder
	store    *milvus.Store
	cfg      *Config
}

func NewModelService(cfg *Config) *ModelService {
	return &ModelService{
		cfg: cfg,
	}
}

func (s *ModelService) initModels() (llms.Model, embeddings.Embedder, error) {
	llm, err := ollama.New(
		ollama.WithModel(s.cfg.Ollama.LLMModel),
		ollama.WithServerURL(s.cfg.Ollama.Address),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	embedderModel, err := ollama.New(
		ollama.WithModel(s.cfg.Ollama.EmbedderModel),
		ollama.WithServerURL(s.cfg.Ollama.Address),
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

func (s *ModelService) initVectorStore(ctx context.Context, embedder embeddings.Embedder) (*milvus.Store, error) {
	idx, err := entity.NewIndexIvfFlat(entity.L2, 128)
	if err != nil {
		return nil, fmt.Errorf("failed to create ivf flat index: %w", err)
	}

	store, err := milvus.New(
		ctx,
		client.Config{
			Address: s.cfg.Milvus.Address,
			DBName:  s.cfg.Milvus.DBName,
		},
		milvus.WithEmbedder(embedder),
		milvus.WithCollectionName(s.cfg.Milvus.Collection),
		milvus.WithIndex(idx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Milvus store: %w", err)
	}
	// docs := utils.TextToChunks("./index.txt", 50, 0)
	// _, err = store.AddDocuments(ctx, docs)
	// if err != nil {
	// 	log.Fatalf("AddDocument: %v\n", err)
	// }
	return &store, nil
}
