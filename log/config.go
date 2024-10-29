package log

type Config struct {
	//日志存储类型:stdout、file、kafka
	Type        string `yaml:"type"`
	LogFileName string `yaml:"log_file_name"`
	LogFilePath string `yaml:"log_file_path"`
	MaxSize     int64  `yaml:"max_size"`   //最大存储空间单位（MB）
	MaxBackups  int    `yaml:"max_backup"` //最大文件个数
	MaxAge      int    `yaml:"max_age"`    //最大天数
	// 日志级别
	LogLevel int `yaml:"log_level"`
	//日志格式: text,json
	Encoding string `yaml:"encoding"`
	Topic    string `yaml:"topic"`
}
