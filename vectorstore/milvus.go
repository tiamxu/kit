package vectorstore

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/milvus"
)

func (m *MilvusConfig) Validate() error {
	if m.Address == "" {
		return fmt.Errorf("milvus address is required")
	}
	if m.Collection == "" {
		return fmt.Errorf("milvus collection name is required")
	}
	if m.Index.Type == "" {
		return fmt.Errorf("milvus index type is required")
	}
	return nil
}

type MilvusStore struct {
	store    *milvus.Store
	embedder embeddings.Embedder
	cfg      *MilvusConfig
}

func NewMilvusStore(cfg *MilvusConfig, embedder embeddings.Embedder) *MilvusStore {
	return &MilvusStore{
		embedder: embedder,
		cfg:      cfg,
	}
}

func (m *MilvusStore) Initialize(ctx context.Context) error {
	if err := m.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	var idx entity.Index
	var err error

	switch m.cfg.Index.Type {
	case "IVF_FLAT":
		idx, err = entity.NewIndexIvfFlat(entity.L2, 768)
	case "IVF_SQ8":
		idx, err = entity.NewIndexIvfSQ8(entity.L2, 768)
	case "HNSW":
		idx, err = entity.NewIndexHNSW(entity.L2, 16, 200)
	default:
		return fmt.Errorf("unsupported index type: %s", m.cfg.Index.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	store, err := milvus.New(
		ctx,
		client.Config{
			Address: m.cfg.Address,
			DBName:  m.cfg.DBName,
		},
		milvus.WithEmbedder(m.embedder),
		milvus.WithCollectionName(m.cfg.Collection),
		milvus.WithIndex(idx),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Milvus store: %w", err)
	}

	m.store = &store
	return nil
}

func (m *MilvusStore) CreateCollection(ctx context.Context) error {
	if m.store == nil {
		return fmt.Errorf("store not initialized")
	}
	// Create Milvus client
	milvusClient, err := client.NewClient(ctx, client.Config{
		Address: m.cfg.Address,
		DBName:  m.cfg.DBName,
	})
	if err != nil {
		return fmt.Errorf("failed to create Milvus client: %w", err)
	}
	// Define collection schema
	schema := &entity.Schema{
		CollectionName: m.cfg.Collection,
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "text",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "meta",
				DataType: entity.FieldTypeJSON,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": "768",
				},
			},
		},
	}
	has, err := milvusClient.HasCollection(context.Background(), m.cfg.Collection)
	if err != nil {
		return fmt.Errorf("failed to check collection exists: %w", err)
	}
	if !has {
		err = milvusClient.CreateCollection(ctx, schema, 1) // 2 shards
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}
func (m *MilvusStore) AddDocuments(ctx context.Context, docs []schema.Document) error {
	_, err := m.store.AddDocuments(ctx, docs)
	return err
}

// func (m *MilvusStore) SimilaritySearch(ctx context.Context, query string, k int) ([]schema.Document, error) {
// 	return m.store.SimilaritySearch(ctx, query, k)
// }

func (m *MilvusStore) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.store.SimilaritySearch(ctx, query, k)
}

func (m *MilvusStore) Close(ctx context.Context) error {
	return nil
}
