package redis

const (
	// CacheFormatRaw raw
	CacheFormatRaw = 0
	// CacheFormatRawGzip raw gzip
	CacheFormatRawGzip = 1
	// CacheFormatJSON json
	CacheFormatJSON = 10
	// CacheFormatJSONGzip json gzip
	CacheFormatJSONGzip = 11
)

type Config struct {
	// redis服务器地址，ip:port格式 默认为 :6379
	Address string `yaml:"address" json:"address"`
	// 默认为空，不进行认证
	Password string `yaml:"password" json:"password"`
	// redis DB 数据库，默认为0
	DB int `yaml:"db" json:"db"`
	//连接池最大连接数量,默认为 10 * runtime.GOMAXPROCS
	PoolSize int `yaml:"pool_size"`
	// 连接池保持的最大空闲连接数，多余的空闲连接将被关闭,默认为0，不限制
	MaxIdle int `yaml:"max_idle" json:"max_idle"`
	// 连接池保持的最小空闲连接数，它受到PoolSize的限制,默认为0，不限制
	MinIdle int `yaml:"min_idle" json:"min_idle"`
	// DialTimeout 连接建立超时时间，默认5秒。
	DialTimeout int `yaml:"dial_timeout"`
	// Timeout 读写超时时间, 默认3秒， -1表示取消读超时
	Timeout int `yaml:"timeout" json:"timeout"`
	// IdleTimeout 闲置超时，默认5分钟，-1表示取消闲置超时检查
	IdleTimeout int `yaml:"idle_timeout" json:"idle_timeout"`
	RetryTimes  int `yaml:"retry_times" json:"retry_times"`
	// 压缩阈值配置
	GzipMinSize int `yaml:"gzip_min_size" json:"gzip_min_size"`
}

type modelCacheItem struct {
	Flag uint32
	Data []byte
}
