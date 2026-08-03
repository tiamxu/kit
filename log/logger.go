package log

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultMaxSize    = 100
	defaultMaxBackups = 5
	defaultMaxAge     = 30
)

type Config struct {
	Level      string `yaml:"level" json:"level"`
	FilePath   string `yaml:"file_path" json:"file_path"`
	FileName   string `yaml:"file_name" json:"file_name"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
	Type       string `yaml:"type" json:"type"`
	Format     string `yaml:"format" json:"format"`
}

type Fields = map[string]any

var (
	Sugar             *zap.SugaredLogger
	Logger            *zap.Logger
	recordSugar       *zap.SugaredLogger
	_fields           Fields
	_logger           atomic.Value // 存储 Logger 接口，用于并发安全
	logFormat         atomic.Value
	loggerInitialized atomic.Bool
)

var (
	ErrLoggerInitialized = errors.New("logger already initialized")
	ErrNilConfig         = errors.New("log config cannot be nil")
)

type loggerBundle struct {
	logger      *zap.Logger
	sugar       *zap.SugaredLogger
	recordSugar *zap.SugaredLogger
	format      string
}

func InitLogger(cfg *Config) error {
	if cfg == nil {
		return ErrNilConfig
	}

	sinks, err := buildWriteSyncers(cfg)
	if err != nil {
		return err
	}
	bundle, err := newLoggerBundle(cfg, sinks)
	if err != nil {
		return err
	}

	if !loggerInitialized.CompareAndSwap(false, true) {
		return ErrLoggerInitialized
	}
	Logger = bundle.logger
	Sugar = bundle.sugar
	recordSugar = bundle.recordSugar
	logFormat.Store(bundle.format)
	_logger.Store(Logger)

	return nil
}

func newLoggerBundle(cfg *Config, sinks []zapcore.WriteSyncer) (*loggerBundle, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format != "json" {
		format = "console"
	}
	if len(sinks) == 0 {
		sinks = []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	}
	enabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level
	})
	normalCores := make([]zapcore.Core, 0, len(sinks))
	recordCores := make([]zapcore.Core, 0, len(sinks))
	for _, sink := range sinks {
		normalCores = append(normalCores, zapcore.NewCore(newEncoder(format, true), sink, enabler))
		recordCores = append(recordCores, zapcore.NewCore(newEncoder(format, false), sink, enabler))
	}
	logger := zap.New(zapcore.NewTee(normalCores...), zap.AddCaller(), zap.AddCallerSkip(1))
	recordLogger := zap.New(zapcore.NewTee(recordCores...), zap.AddCaller(), zap.AddCallerSkip(1))
	return &loggerBundle{
		logger:      logger,
		sugar:       logger.Sugar(),
		recordSugar: recordLogger.Sugar(),
		format:      format,
	}, nil
}

func newEncoder(format string, includeMessage bool) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if !includeMessage {
		encoderConfig.MessageKey = zapcore.OmitKey
	}
	if format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func (b *loggerBundle) infoRecord(fields Fields, consoleText string) {
	if b.format == "json" {
		b.recordSugar.With(fieldsToArgs(fields)...).Info()
		return
	}
	b.sugar.Info(consoleText)
}

func buildWriteSyncers(cfg *Config) ([]zapcore.WriteSyncer, error) {
	outputType := strings.ToLower(cfg.Type)
	parts := strings.Split(outputType, "+")
	sinks := make([]zapcore.WriteSyncer, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "stdout", "":
			sinks = append(sinks, zapcore.AddSync(os.Stdout))
		case "file":
			fileSink, err := buildFileWriteSyncer(cfg)
			if err != nil {
				return nil, fmt.Errorf("setup file output: %w", err)
			}
			sinks = append(sinks, fileSink)
		default:
			return nil, fmt.Errorf("unknown output type: %s", part)
		}
	}
	if len(sinks) == 0 {
		sinks = append(sinks, zapcore.AddSync(os.Stdout))
	}
	return sinks, nil
}

func buildFileWriteSyncer(cfg *Config) (zapcore.WriteSyncer, error) {
	if cfg.FilePath == "" {
		cfg.FilePath = "logs"
	}
	if cfg.FileName == "" {
		cfg.FileName = "app.log"
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = defaultMaxSize
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = defaultMaxBackups
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = defaultMaxAge
	}

	if err := os.MkdirAll(cfg.FilePath, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	filePath := filepath.Join(cfg.FilePath, cfg.FileName)
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	return zapcore.AddSync(writer), nil
}

func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug", "debug1":
		return zap.DebugLevel, nil
	case "info", "info1", "notice", "trace":
		return zap.InfoLevel, nil
	case "warn", "warning", "warn1":
		return zap.WarnLevel, nil
	case "error", "err", "error1":
		return zap.ErrorLevel, nil
	case "fatal", "critical", "crit":
		return zap.FatalLevel, nil
	case "panic":
		return zap.PanicLevel, nil
	default:
		return zap.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

var _once sync.Once

func ensureLogger() {
	if _logger.Load() != nil {
		return
	}
	_once.Do(func() {
		if _logger.Load() != nil {
			return
		}
		bundle, err := newLoggerBundle(&Config{Level: "info", Format: "console"}, []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)})
		if err != nil {
			return
		}
		Logger = bundle.logger
		Sugar = bundle.sugar
		recordSugar = bundle.recordSugar
		logFormat.Store(bundle.format)
		_logger.Store(Logger)
	})
}

func GetLogger() *zap.SugaredLogger {
	if l := _logger.Load(); l != nil {
		return l.(*zap.Logger).Sugar()
	}
	ensureLogger()
	return Sugar
}

// SetGlobalFields 设置全局日志字段，建议仅在启动阶段调用。
func SetGlobalFields(fields Fields) {
	_fields = fields
}

func getMergedFields(fields Fields) []any {
	if len(_fields) == 0 {
		return fieldsToArgs(fields)
	}
	merged := make(Fields, len(_fields)+len(fields))
	for k, v := range _fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return fieldsToArgs(merged)
}

func fieldsToArgs(fields Fields) []any {
	if len(fields) == 0 {
		return nil
	}
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

type SugaredEntry struct {
	sugar *zap.SugaredLogger
}

func WithFields(fields Fields) *SugaredEntry {
	ensureLogger()
	args := getMergedFields(fields)
	return &SugaredEntry{sugar: Sugar.With(args...)}
}

func InfoRecord(fields Fields, consoleText string) {
	ensureLogger()
	format, _ := logFormat.Load().(string)
	if format == "json" {
		recordSugar.With(getMergedFields(fields)...).Info()
		return
	}
	Sugar.Info(consoleText)
}

func WithContext(ctx context.Context) *SugaredEntry {
	ensureLogger()
	args := []any{}
	if requestID := ctx.Value("request_id"); requestID != nil {
		args = append(args, "request_id", requestID)
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		args = append(args, "trace_id", traceID)
	}
	return &SugaredEntry{sugar: Sugar.With(args...)}
}

func (e *SugaredEntry) Info(msg string)                   { e.sugar.Infow(msg) }
func (e *SugaredEntry) Warn(msg string)                   { e.sugar.Warnw(msg) }
func (e *SugaredEntry) Error(msg string)                  { e.sugar.Errorw(msg) }
func (e *SugaredEntry) Debug(msg string)                  { e.sugar.Debugw(msg) }
func (e *SugaredEntry) Fatal(msg string)                  { e.sugar.Fatalw(msg) }
func (e *SugaredEntry) Panic(msg string)                  { e.sugar.Panicw(msg) }
func (e *SugaredEntry) Infof(format string, args ...any)  { e.sugar.Infof(format, args...) }
func (e *SugaredEntry) Warnf(format string, args ...any)  { e.sugar.Warnf(format, args...) }
func (e *SugaredEntry) Errorf(format string, args ...any) { e.sugar.Errorf(format, args...) }
func (e *SugaredEntry) Debugf(format string, args ...any) { e.sugar.Debugf(format, args...) }
func (e *SugaredEntry) Fatalf(format string, args ...any) { e.sugar.Fatalf(format, args...) }
func (e *SugaredEntry) Panicf(format string, args ...any) { e.sugar.Panicf(format, args...) }

func Tracef(format string, args ...any) {
	ensureLogger()
	Sugar.Debugf(format, args...)
}

func Traceln(args ...any) {
	ensureLogger()
	Sugar.Debug(args...)
}

func Debugf(format string, args ...any) {
	ensureLogger()
	Sugar.Debugf(format, args...)
}

func Debugln(args ...any) {
	ensureLogger()
	Sugar.Debug(args...)
}

func Printf(format string, args ...any) {
	ensureLogger()
	Sugar.Infof(format, args...)
}

func Println(args ...any) {
	ensureLogger()
	Sugar.Info(args...)
}

func Infof(format string, args ...any) {
	ensureLogger()
	Sugar.Infof(format, args...)
}

func Infoln(args ...any) {
	ensureLogger()
	Sugar.Info(args...)
}

func Warnf(format string, args ...any) {
	ensureLogger()
	Sugar.Warnf(format, args...)
}

func Warnln(args ...any) {
	ensureLogger()
	Sugar.Warn(args...)
}

func Errorf(format string, args ...any) {
	ensureLogger()
	Sugar.Errorf(format, args...)
}

func Errorln(args ...any) {
	ensureLogger()
	Sugar.Error(args...)
}

func Panicf(format string, args ...any) {
	ensureLogger()
	Sugar.Panicf(format, args...)
}

func Panicln(args ...any) {
	ensureLogger()
	Sugar.Panic(args...)
}

func Fatalf(format string, args ...any) {
	ensureLogger()
	Sugar.Fatalf(format, args...)
}

func Fatalln(args ...any) {
	ensureLogger()
	Sugar.Fatal(args...)
}

func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
