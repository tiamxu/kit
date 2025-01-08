package log

// type Config struct {
// 	//日志存储类型:stdout、file、kafka
// 	Type        string `yaml:"type"`
// 	LogFileName string `yaml:"log_file_name"`
// 	LogFilePath string `yaml:"log_file_path"`
// 	MaxSize     int64  `yaml:"max_size"`   //最大存储空间单位（MB）
// 	MaxBackups  int    `yaml:"max_backup"` //最大文件个数
// 	MaxAge      int    `yaml:"max_age"`    //最大天数
// 	// 日志级别
// 	LogLevel string `yaml:"log_level"`
// 	//日志格式: text,json
// 	Format string `yaml:"format"`
// 	Topic  string `yaml:"topic"`
// }

type Config struct {
	LogLevel    string `json:"log_level"`
	LogFilePath string `json:"log_file_path"`
	LogFileName string `json:"log_file_name"`
	MaxSize     int    `json:"max_size"`
	MaxBackups  int    `json:"max_backups"`
	MaxAge      int    `json:"max_age"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	KafkaConfig struct {
		Brokers    []string `json:"brokers"`
		Topic      string   `json:"topic"`
		MaxRetries int      `json:"max_retries"`
	} `json:"kafka_config"`
}
