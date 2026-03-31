package vectorstore

import (
	"context"
	"fmt"
	"strconv"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/milvus"
)

const (
	defaultDimension      = 768
	defaultNList          = 128
	defaultM              = 16
	defaultEfConstruction = 200
	defaultMaxLength      = 65535
)

// Validate 校验Milvus配置
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

// MilvusStore Milvus向量存储实现
type MilvusStore struct {
	store    *milvus.Store
	embedder embeddings.Embedder
	cfg      *MilvusConfig
}

// NewMilvusStore 创建Milvus向量存储实例
// 参数:
//
//	cfg: Milvus配置
//	embedder: 向量嵌入器
//
// 返回:
//
//	*MilvusStore: Milvus向量存储实例
func NewMilvusStore(cfg *MilvusConfig, embedder embeddings.Embedder) *MilvusStore {
	return &MilvusStore{
		embedder: embedder,
		cfg:      cfg,
	}
}

// Initialize 初始化Milvus向量存储，包含自动创建集合
// 参数:
//
//	ctx: 上下文
//
// 返回:
//
//	error: 错误信息
func (m *MilvusStore) Initialize(ctx context.Context) error {
	if err := m.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	dimension := m.cfg.Dimension
	if dimension <= 0 {
		dimension = defaultDimension
	}

	metricType := parseMetricType(m.cfg.Index.MetricType)

	var idx entity.Index
	var err error

	switch m.cfg.Index.Type {
	case "IVF_FLAT":
		nList := m.cfg.Index.NList
		if nList <= 0 {
			nList = defaultNList
		}
		idx, err = entity.NewIndexIvfFlat(metricType, nList)
	case "IVF_SQ8":
		nList := m.cfg.Index.NList
		if nList <= 0 {
			nList = defaultNList
		}
		idx, err = entity.NewIndexIvfSQ8(metricType, nList)
	case "HNSW":
		idx, err = entity.NewIndexHNSW(metricType, defaultM, defaultEfConstruction)
	default:
		return fmt.Errorf("unsupported index type: %s", m.cfg.Index.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	store, err := milvus.New(
		ctx,
		client.Config{
			Address:  m.cfg.Address,
			DBName:   m.cfg.DBName,
			Username: m.cfg.Username,
			Password: m.cfg.Password,
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

// createCollection 创建Milvus集合
func (m *MilvusStore) createCollection(ctx context.Context) error {
	milvusClient, err := client.NewClient(ctx, client.Config{
		Address:  m.cfg.Address,
		DBName:   m.cfg.DBName,
		Username: m.cfg.Username,
		Password: m.cfg.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create Milvus client: %w", err)
	}
	defer milvusClient.Close()

	dimension := m.cfg.Dimension
	if dimension <= 0 {
		dimension = defaultDimension
	}

	maxLength := m.cfg.MaxLength
	if maxLength <= 0 {
		maxLength = defaultMaxLength
	}

	collectionSchema := &entity.Schema{
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
					"max_length": strconv.Itoa(maxLength),
				},
			},
			{
				Name:     "meta",
				DataType: entity.FieldTypeJSON,
				TypeParams: map[string]string{
					"max_length": strconv.Itoa(maxLength),
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": strconv.Itoa(dimension),
				},
			},
		},
	}

	has, err := milvusClient.HasCollection(ctx, m.cfg.Collection)
	if err != nil {
		return fmt.Errorf("failed to check collection exists: %w", err)
	}
	if !has {
		if err := milvusClient.CreateCollection(ctx, collectionSchema, 1); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

// AddDocuments 添加文档到Milvus
// 参数:
//
//	ctx: 上下文
//	docs: 文档列表
//
// 返回:
//
//	error: 错误信息
func (m *MilvusStore) AddDocuments(ctx context.Context, docs []schema.Document) error {
	_, err := m.store.AddDocuments(ctx, docs)
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
func (m *MilvusStore) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.store.SimilaritySearch(ctx, query, k)
}

// Close 关闭Milvus连接
// 参数:
//
//	ctx: 上下文
//
// 返回:
//
//	error: 错误信息
func (m *MilvusStore) Close(ctx context.Context) error {
	return nil
}

// parseMetricType 解析度量类型配置
func parseMetricType(metricType string) entity.MetricType {
	switch metricType {
	case "IP":
		return entity.IP
	case "COSINE":
		return entity.COSINE
	case "HAMMING":
		return entity.HAMMING
	case "JACCARD":
		return entity.JACCARD
	default:
		return entity.L2
	}
}
