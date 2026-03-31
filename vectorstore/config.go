package vectorstore

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	Type   string       `yaml:"type" json:"type"`
	Milvus MilvusConfig `yaml:"milvus" json:"milvus"`
	Qdrant QdrantConfig `yaml:"qdrant" json:"qdrant"`
}

// MilvusConfig Milvus配置
type MilvusConfig struct {
	Address    string      `yaml:"address" json:"address"`
	DBName     string      `yaml:"db_name" json:"db_name"`
	Collection string      `yaml:"collection" json:"collection"`
	Index      IndexConfig `yaml:"index" json:"index"`
	Username   string      `yaml:"username,omitempty" json:"-"`
	Password   string      `yaml:"password,omitempty" json:"-"`
	Dimension  int         `yaml:"dimension" json:"dimension"`
	MaxLength  int         `yaml:"max_length,omitempty" json:"max_length,omitempty"`
}

// IndexConfig 索引配置
type IndexConfig struct {
	Type       string `yaml:"type" json:"type"`
	MetricType string `yaml:"metric_type" json:"metric_type"`
	NList      int    `yaml:"nlist" json:"nlist"`
}

// QdrantConfig Qdrant配置
type QdrantConfig struct {
	Address    string `yaml:"address" json:"address"`
	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port"`
	Collection string `yaml:"collection" json:"collection"`
	ApiKey     string `yaml:"api_key" json:"-"`
	Dimension  int    `yaml:"dimension" json:"dimension"`
	Distance   string `yaml:"distance" json:"distance"`
	Https      bool   `yaml:"https" json:"https"`
}
