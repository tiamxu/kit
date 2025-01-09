package log

import (
	"context"
	"os"
	"sync"
	"time"

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
