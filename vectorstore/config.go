package vectorstore

type VectorStoreConfig struct {
	Type   string       `yaml:"type"`
	Milvus MilvusConfig `yaml:"milvus"`
	Qdrant QdrantConfig `yaml:"qdrant"`
}
type MilvusConfig struct {
	Address    string      `yaml:"address"`
	DBName     string      `yaml:"db_name"`
	Collection string      `yaml:"collection"`
	Index      IndexConfig `yaml:"index"`
}

type IndexConfig struct {
	Type       string `yaml:"type"`
	MetricType string `yaml:"metric_type"`
	NList      int    `yaml:"nlist"`
}

type QdrantConfig struct {
	Address    string `yaml:"address"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Collection string `yaml:"collection"`
	ApiKey     string `yaml:"api_key"`
}
