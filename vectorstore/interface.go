package vectorstore

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
)

// NewVectorStore 工厂函数，根据配置创建对应的向量存储实例
func NewVectorStore(cfg *VectorStoreConfig, embedder embeddings.Embedder) (VectorStore, error) {
	switch cfg.Type {
	case "milvus":
		store := NewMilvusStore(&cfg.Milvus, embedder)
		return store, nil
	case "qdrant":
		store := NewQdrantStore(&cfg.Qdrant, embedder)
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", cfg.Type)
	}
}

// VectorStore 定义向量存储的通用接口
type VectorStore interface {
	// Initialize 初始化向量存储
	Initialize(ctx context.Context) error

	// CreateCollection 创建集合
	CreateCollection(ctx context.Context) error

	// DeleteCollection 删除集合
	// DeleteCollection(ctx context.Context) error

	// AddDocuments 添加文档到向量存储
	AddDocuments(ctx context.Context, docs []schema.Document) error

	// UpdateDocuments 更新文档
	// UpdateDocuments(ctx context.Context, docs []schema.Document) error

	// DeleteDocuments 删除文档
	// DeleteDocuments(ctx context.Context, ids []string) error

	// Search 执行相似度搜索
	Search(ctx context.Context, query string, topK int) ([]schema.Document, error)

	// GetDocumentsByID 根据ID获取文档
	// GetDocumentsByID(ctx context.Context, ids []string) ([]schema.Document, error)

	// Close 关闭向量存储连接
	Close(ctx context.Context) error
}

// Config 向量存储配置接口
type Config interface {
	Validate() error
}
