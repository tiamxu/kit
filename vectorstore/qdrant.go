package vectorstore

import (
	"context"
	"fmt"
	"net/url"

	client "github.com/qdrant/go-client/qdrant"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

type QdrantStore struct {
	store    *qdrant.Store
	embedder embeddings.Embedder
	cfg      *QdrantConfig
}

func (c *QdrantConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("qdrant address is required")
	}
	if c.Collection == "" {
		return fmt.Errorf("qdrant collection name is required")
	}
	return nil
}

func NewQdrantStore(cfg *QdrantConfig, embedder embeddings.Embedder) *QdrantStore {
	return &QdrantStore{
		embedder: embedder,
		cfg:      cfg,
	}
}

func (q *QdrantStore) Initialize(ctx context.Context) error {
	qdrantURL, err := url.Parse(q.cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid Qdrant URL: %w", err)
	}

	store, err := qdrant.New(
		qdrant.WithURL(*qdrantURL),
		qdrant.WithCollectionName(q.cfg.Collection),
		qdrant.WithEmbedder(q.embedder),
		qdrant.WithAPIKey("ZfYOjrdr2io25WUKvpdwnJ8gfvc"),
	)

	if err != nil {
		return fmt.Errorf("failed to initialize Qdrant store: %w", err)
	}

	q.store = &store
	return nil
}
func (q *QdrantStore) CreateCollection(ctx context.Context) error {
	qdrantClient, err := client.NewClient(&client.Config{
		Host:                   q.cfg.Address,
		Port:                   q.cfg.Port,
		APIKey:                 q.cfg.ApiKey,
		UseTLS:                 false,
		SkipCompatibilityCheck: true,
	})

	if err != nil {
		return fmt.Errorf("连接qdrant服务器失败: %w", err)
	}
	ok, err := qdrantClient.CollectionExists(ctx, q.cfg.Collection)
	if err != nil {
		return fmt.Errorf("无法检查集合是否存在: %w", err)
	}
	if !ok {
		err = qdrantClient.CreateCollection(ctx, &client.CreateCollection{
			CollectionName: q.cfg.Collection,
			VectorsConfig: client.NewVectorsConfig(&client.VectorParams{
				Size:     768,
				Distance: client.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("创建集合失败: %w", err)
		}
	}
	return nil
}
func (q *QdrantStore) AddDocuments(ctx context.Context, docs []schema.Document) error {
	_, err := q.store.AddDocuments(ctx, docs)
	return err
}

// func (q *QdrantStore) SimilaritySearch(ctx context.Context, query string, k int) ([]schema.Document, error) {
// 	return q.store.SimilaritySearch(ctx, query, k)
// }

func (q *QdrantStore) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return q.store.SimilaritySearch(ctx, query, k)
}

func (q *QdrantStore) Close(ctx context.Context) error {
	return nil
}
