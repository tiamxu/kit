package vector

type Config struct {
	Milvus     MilvusConfig     `yaml:"milvus"`
	Ollama     OllamaConfig     `yaml:"ollama"`
	Processing ProcessingConfig `yaml:"processing"`
}

type MilvusConfig struct {
	Address    string      `yaml:"address"`
	DBName     string      `yaml:"db_name"`
	Collection string      `yaml:"collection"`
	Index      IndexConfig `yaml:"index"`
}

type OllamaConfig struct {
	Address       string  `yaml:"address"`
	LLMModel      string  `yaml:"llm_model"`
	EmbedderModel string  `yaml:"embedder_model"`
	Temperature   float64 `yaml:"temperature"`
}

type ProcessingConfig struct {
	ChunkSize      int     `yaml:"chunk_size"`
	ChunkOverlap   int     `yaml:"chunk_overlap"`
	TopK           int     `yaml:"top_k"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

type IndexConfig struct {
	Type       string `yaml:"type"`
	MetricType string `yaml:"metric_type"`
	NList      int    `yaml:"nlist"`
}
