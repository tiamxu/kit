package vectorstore

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
)

// NewVectorStore 工厂函数，根据配置创建对应的向量存储实例
// 参数:
//
//	cfg: 向量存储配置
//	embedder: 向量嵌入器
//
// 返回:
//
//	VectorStore: 向量存储实例
//	error: 错误信息
func NewVectorStore(cfg *VectorStoreConfig, embedder embeddings.Embedder) (VectorStore, error) {
	switch cfg.Type {
	case "milvus":
		return NewMilvusStore(&cfg.Milvus, embedder), nil
	case "qdrant":
		return NewQdrantStore(&cfg.Qdrant, embedder), nil
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", cfg.Type)
	}
}

// VectorStore 定义向量存储的通用接口
type VectorStore interface {
	// Initialize 初始化向量存储（含自动创建集合）
	Initialize(ctx context.Context) error

	// AddDocuments 添加文档到向量存储
	AddDocuments(ctx context.Context, docs []schema.Document) error

	// Search 执行相似度搜索
	Search(ctx context.Context, query string, topK int) ([]schema.Document, error)

	// Close 关闭向量存储连接
	Close(ctx context.Context) error
}
