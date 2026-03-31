package redis

// Config Redis客户端配置
type Config struct {
	Address string `yaml:"address" json:"address"`
	Password string `yaml:"password" json:"-"`
	DB       int    `yaml:"db" json:"db"`
	PoolSize int    `yaml:"pool_size" json:"pool_size"`
	MaxIdle  int    `yaml:"max_idle" json:"max_idle"`
	MinIdle  int    `yaml:"min_idle" json:"min_idle"`
	DialTimeout int `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  int `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout" json:"write_timeout"`
}
