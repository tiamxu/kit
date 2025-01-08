package log

import (
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
		_defaultLogger.SetReportCaller(true)
	})
	return _defaultLogger
}

func Init(cfg *Config) error {
	logger := DefaultLogger()

	// Set log level
	level := logrus.Level(cfg.LogLevel)
	logger.SetLevel(level)

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
		// TODO: implement kafka output
		logger.SetOutput(os.Stdout)
	default:
		logger.SetOutput(os.Stdout)
	}

	// Set formatter based on encoding
	if cfg.Encoding == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	return nil
}

type Fields = logrus.Fields

var WithFields = _defaultLogger.WithFields
var WithContext = _defaultLogger.WithContext
var Traceln = _defaultLogger.Traceln
var Tracef = _defaultLogger.Tracef
var Debugf = _defaultLogger.Debugf
var Debugln = _defaultLogger.Debugln
var Printf = _defaultLogger.Printf
var Println = _defaultLogger.Println
var Infof = _defaultLogger.Infof
var Infoln = _defaultLogger.Infoln
var Warnf = _defaultLogger.Warnf
var Warnln = _defaultLogger.Warnln
var Errorf = _defaultLogger.Errorf
var Errorln = _defaultLogger.Errorln
var Panicf = _defaultLogger.Panicf
var Paincln = _defaultLogger.Panicln
var Fatalf = _defaultLogger.Fatalf
var Fatalln = _defaultLogger.Fatalln
