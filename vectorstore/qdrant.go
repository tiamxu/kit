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

const defaultQdrantDimension = 768

// Validate 校验Qdrant配置
func (c *QdrantConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("qdrant address is required")
	}
	if c.Collection == "" {
		return fmt.Errorf("qdrant collection name is required")
	}
	return nil
}

// QdrantStore Qdrant向量存储实现
type QdrantStore struct {
	store    *qdrant.Store
	embedder embeddings.Embedder
	cfg      *QdrantConfig
}

// NewQdrantStore 创建Qdrant向量存储实例
// 参数:
//
//	cfg: Qdrant配置
//	embedder: 向量嵌入器
//
// 返回:
//
//	*QdrantStore: Qdrant向量存储实例
func NewQdrantStore(cfg *QdrantConfig, embedder embeddings.Embedder) *QdrantStore {
	return &QdrantStore{
		embedder: embedder,
		cfg:      cfg,
	}
}

// Initialize 初始化Qdrant向量存储，包含自动创建集合
// 参数:
//
//	ctx: 上下文
//
// 返回:
//
//	error: 错误信息
func (q *QdrantStore) Initialize(ctx context.Context) error {
	if err := q.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	qdrantURL, err := url.Parse(q.cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid Qdrant URL: %w", err)
	}

	if err := q.ensureCollection(ctx); err != nil {
		return err
	}

	store, err := qdrant.New(
		qdrant.WithURL(*qdrantURL),
		qdrant.WithCollectionName(q.cfg.Collection),
		qdrant.WithEmbedder(q.embedder),
		qdrant.WithAPIKey(q.cfg.ApiKey),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Qdrant store: %w", err)
	}

	q.store = &store
	return nil
}

// ensureCollection 确保Qdrant集合存在，不存在则创建
func (q *QdrantStore) ensureCollection(ctx context.Context) error {
	qdrantClient, err := client.NewClient(&client.Config{
		Host:                   q.cfg.Host,
		Port:                   q.cfg.Port,
		APIKey:                 q.cfg.ApiKey,
		UseTLS:                 q.cfg.Https,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}
	defer qdrantClient.Close()

	ok, err := qdrantClient.CollectionExists(ctx, q.cfg.Collection)
	if err != nil {
		return fmt.Errorf("failed to check collection exists: %w", err)
	}
	if !ok {
		dimension := q.cfg.Dimension
		if dimension <= 0 {
			dimension = defaultQdrantDimension
		}

		distance := parseDistance(q.cfg.Distance)

		err = qdrantClient.CreateCollection(ctx, &client.CreateCollection{
			CollectionName: q.cfg.Collection,
			VectorsConfig: client.NewVectorsConfig(&client.VectorParams{
				Size:     uint64(dimension),
				Distance: distance,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

// AddDocuments 添加文档到Qdrant
// 参数:
//
//	ctx: 上下文
//	docs: 文档列表
//
// 返回:
//
//	error: 错误信息
func (q *QdrantStore) AddDocuments(ctx context.Context, docs []schema.Document) error {
	_, err := q.store.AddDocuments(ctx, docs)
	return err
}

// Search 执行相似度搜索
// 参数:
//
//	ctx: 上下文
//	query: 查询文本
//	k: 返回结果数量
//
// 返回:
//
//	[]schema.Document: 匹配的文档列表
//	error: 错误信息
func (q *QdrantStore) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return q.store.SimilaritySearch(ctx, query, k)
}

// Close 关闭Qdrant连接
// 参数:
//
//	ctx: 上下文
//
// 返回:
//
//	error: 错误信息
func (q *QdrantStore) Close(ctx context.Context) error {
	return nil
}

// parseDistance 解析距离度量类型配置
func parseDistance(distance string) client.Distance {
	switch distance {
	case "Euclid":
		return client.Distance_Euclid
	case "Dot":
		return client.Distance_Dot
	case "Manhattan":
		return client.Distance_Manhattan
	default:
		return client.Distance_Cosine
	}
}
