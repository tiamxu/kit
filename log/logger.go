package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Level      string `yaml:"level"`
	FilePath   string `yaml:"file_path"`
	FileName   string `yaml:"file_name"`
	MaxSize    int    `yaml:"max_size"`    //最大存储空间单位（MB）
	MaxBackups int    `yaml:"max_backups"` //最大文件个数
	MaxAge     int    `yaml:"max_age"`     //最大天数
	Compress   bool   `yaml:"compress"`
	Type       string `yaml:"type"`   //日志存储类型:stdout、file、kafka
	Format     string `yaml:"format"` //日志格式: text,json
}

var (
	_defaultLogger *logrus.Logger
	once           sync.Once
)

const defaultTimestampFormat = time.RFC3339

// InitLogger 初始化日志配置
func InitLogger(cfg *Config) error {
	once.Do(func() {
		_defaultLogger = logrus.New()

		// 设置日志级别
		level, err := logrus.ParseLevel(cfg.Level)
		if err != nil {
			level = logrus.InfoLevel
		}
		_defaultLogger.SetLevel(level)

		// 设置日志格式
		if cfg.Format == "json" {
			_defaultLogger.SetFormatter(&logrus.JSONFormatter{
				TimestampFormat: defaultTimestampFormat,
			})
		} else {
			_defaultLogger.SetFormatter(&logrus.TextFormatter{
				TimestampFormat: defaultTimestampFormat,
				FullTimestamp:   true,
			})
		}

		// 设置输出
		var output io.Writer
		switch cfg.Type {
		case "file":
			if output, err = setupFileOutput(cfg); err != nil {
				fmt.Printf("Failed to setup file output: %v, fallback to stdout\n", err)
				output = os.Stdout
			}
		case "stdout", "":
			output = os.Stdout
		default:
			output = os.Stdout
		}
		_defaultLogger.SetOutput(output)
	})
	return nil
}

// setupFileOutput 设置文件输出
func setupFileOutput(cfg *Config) (io.Writer, error) {
	if cfg.FilePath == "" {
		cfg.FilePath = "logs"
	}
	if cfg.FileName == "" {
		cfg.FileName = "app.log"
	}

	// 确保日志目录存在
	if err := os.MkdirAll(cfg.FilePath, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %w", err)
	}

	fileName := filepath.Join(cfg.FilePath, cfg.FileName)
	// 使用 lumberjack 进行日志轮转
	return &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    cfg.MaxSize,    // 每个文件最大尺寸，单位 MB
		MaxBackups: cfg.MaxBackups, // 保留的旧文件个数
		MaxAge:     cfg.MaxAge,     // 保留的天数
		Compress:   cfg.Compress,   // 是否压缩
	}, nil

	// 设置日志轮转时间
	// options := []rotatelogs.Option{
	// 	rotatelogs.WithMaxAge(time.Duration(cfg.MaxAge) * 24 * time.Hour),
	// 	rotatelogs.WithRotationTime(24 * time.Hour),
	// }

	// if cfg.MaxBackups > 0 {
	// 	options = append(options, rotatelogs.WithRotationCount(uint(cfg.MaxBackups)))
	// }

	// writer, err := rotatelogs.New(
	// 	logPath+".%Y%m%d",
	// 	options...,
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("create rotate logs failed: %w", err)
	// }

	// return writer, nil
}

// GetLogger 获取logger实例
func GetLogger() *logrus.Logger {
	if _defaultLogger == nil {
		DefaultLogger()
	}
	return _defaultLogger
}

// 添加一个新的方法来设置全局字段
func SetGlobalFields(fields Fields) {
	if _defaultLogger == nil {
		DefaultLogger()
	}
	_defaultLogger.AddHook(&globalFieldsHook{fields: fields})
}

// globalFieldsHook 用于添加全局字段
type globalFieldsHook struct {
	fields Fields
}

func (h *globalFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (h *globalFieldsHook) Fire(entry *logrus.Entry) error {
	for k, v := range h.fields {
		if _, exists := entry.Data[k]; !exists {
			entry.Data[k] = v
		}
	}
	return nil
}
func DefaultLogger() *logrus.Logger {
	once.Do(func() {
		_defaultLogger = logrus.New()
		_defaultLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
		_defaultLogger.SetOutput(os.Stdout)
		_defaultLogger.SetLevel(logrus.TraceLevel)
		// _defaultLogger.SetReportCaller(true)
	})
	return _defaultLogger
}

type Fields = logrus.Fields

func WithFields(fields Fields) *logrus.Entry {
	return _defaultLogger.WithFields(fields)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return _defaultLogger.WithContext(ctx)
}

func Traceln(args ...interface{}) {
	_defaultLogger.Traceln(args...)
}

func Tracef(format string, args ...interface{}) {
	_defaultLogger.Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	_defaultLogger.Debugf(format, args...)
}

func Debugln(args ...interface{}) {
	_defaultLogger.Debugln(args...)
}

func Printf(format string, args ...interface{}) {
	_defaultLogger.Printf(format, args...)
}

func Println(args ...interface{}) {
	_defaultLogger.Println(args...)
}

func Infof(format string, args ...interface{}) {
	_defaultLogger.Infof(format, args...)
}

func Infoln(args ...interface{}) {
	_defaultLogger.Infoln(args...)
}

func Warnf(format string, args ...interface{}) {
	_defaultLogger.Warnf(format, args...)
}

func Warnln(args ...interface{}) {
	_defaultLogger.Warnln(args...)
}

func Errorf(format string, args ...interface{}) {
	_defaultLogger.Errorf(format, args...)
}

func Errorln(args ...interface{}) {
	_defaultLogger.Errorln(args...)
}

func Panicf(format string, args ...interface{}) {
	_defaultLogger.Panicf(format, args...)
}

func Panicln(args ...interface{}) {
	_defaultLogger.Panicln(args...)
}

func Fatalf(format string, args ...interface{}) {
	_defaultLogger.Fatalf(format, args...)
}

func Fatalln(args ...interface{}) {
	_defaultLogger.Fatalln(args...)
}
