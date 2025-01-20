package vectorstore

import (
	"context"

	"github.com/tmc/langchaingo/schema"
)

// VectorStore 定义向量存储的通用接口
type VectorStore interface {
	// Initialize 初始化向量存储
	Initialize(ctx context.Context) error

	// AddDocuments 添加文档到向量存储
	AddDocuments(ctx context.Context, docs []schema.Document) error

	// Search 执行相似度搜索
	Search(ctx context.Context, query string, topK int) ([]schema.Document, error)

	// Close 关闭向量存储连接
	Close(ctx context.Context) error
}

// Config 向量存储配置接口
type Config interface {
	Validate() error
}
