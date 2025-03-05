package es

type Config struct {
	// ES节点地址
	Addresses []string `yaml:"addresses"`
	// 认证用户名
	Username string `yaml:"username"`
	// 认证密码
	Password string `yaml:"password"`
	// 最大重试次数
	MaxRetries int `yaml:"max_retries"`
	// 请求超时时间
	Timeout     int  `yaml:"timeout"`
	EnableDebug bool `yaml:"enable_debug"` // 启用调试日志
	// 最大空闲连接数
	MaxIdleConns int `yaml:"max_idle_conns"`
	// 空闲连接超时
	IdleConnTimeout int `yaml:"idle_conn_timeout"` // 修正字段名称
}
