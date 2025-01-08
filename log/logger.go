package log

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

var (
	_defaultLogger *logrus.Logger
	once           sync.Once
)

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

func Init(cfg *Config) error {
	logger := DefaultLogger()

	// Set log level
	if level, err := logrus.ParseLevel(cfg.LogLevel); err != nil {
		return err
	} else {
		logger.SetLevel(level)
	}
	// level := logrus.Level(cfg.LogLevel)
	// logger.SetLevel(level)

	// Set output
	switch cfg.Type {
	case "file":
		fileLogger := &lumberjack.Logger{
			Filename:   cfg.LogFilePath + "/" + cfg.LogFileName,
			MaxSize:    int(cfg.MaxSize),
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   true,
		}
		logger.SetOutput(fileLogger)
	case "kafka":
		formatter, writer := setupKafkaOutput(cfg)
		logger.SetFormatter(formatter)
		logger.SetOutput(writer)
	default:
		logger.SetOutput(os.Stdout)
	}

	// Set formatter based on encoding
	if cfg.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
			DisableColors:   true,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	return nil
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
