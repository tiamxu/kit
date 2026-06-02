package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	Sugar       *zap.SugaredLogger
	Logger      *zap.Logger
	_fields     Fields
	globalFormat string // 日志格式：json 或 console
)

func InitLogger(cfg *Config) error {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	outputType := strings.ToLower(cfg.Type)
	format := strings.ToLower(cfg.Format)
	globalFormat = format

	var cores []zapcore.Core

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

	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	enabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level
	})

	parts := strings.Split(outputType, "+")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "stdout", "":
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), enabler))
		case "file":
			fileCore, err := buildFileCore(cfg, encoder, enabler)
			if err != nil {
				return fmt.Errorf("setup file output: %w", err)
			}
			cores = append(cores, fileCore)
		default:
			return fmt.Errorf("unknown output type: %s", part)
		}
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), enabler))
	}

	core := zapcore.NewTee(cores...)
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()

	return nil
}

func buildFileCore(cfg *Config, encoder zapcore.Encoder, enabler zapcore.LevelEnabler) (zapcore.Core, error) {
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

	return zapcore.NewCore(encoder, zapcore.AddSync(writer), enabler), nil
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
	_once.Do(func() {
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:      "time",
			LevelKey:     "level",
			NameKey:      "logger",
			CallerKey:    "caller",
			FunctionKey:  zapcore.OmitKey,
			MessageKey:   "msg",
			StacktraceKey: "stacktrace",
			LineEnding:   zapcore.DefaultLineEnding,
			EncodeLevel:  zapcore.CapitalColorLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		}
		encoder := zapcore.NewConsoleEncoder(encoderConfig)
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.InfoLevel)
		Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		Sugar = Logger.Sugar()
	})
}

func GetGlobalFormat() string {
	return globalFormat
}

func GetLogger() *zap.SugaredLogger {
	ensureLogger()
	return Sugar
}

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

func (e *SugaredEntry) Info(msg string)                           { e.sugar.Infow(msg) }
func (e *SugaredEntry) Warn(msg string)                           { e.sugar.Warnw(msg) }
func (e *SugaredEntry) Error(msg string)                          { e.sugar.Errorw(msg) }
func (e *SugaredEntry) Debug(msg string)                          { e.sugar.Debugw(msg) }
func (e *SugaredEntry) Fatal(msg string)                          { e.sugar.Fatalw(msg) }
func (e *SugaredEntry) Panic(msg string)                          { e.sugar.Panicw(msg) }
func (e *SugaredEntry) Infof(format string, args ...any)         { e.sugar.Infof(format, args...) }
func (e *SugaredEntry) Warnf(format string, args ...any)          { e.sugar.Warnf(format, args...) }
func (e *SugaredEntry) Errorf(format string, args ...any)        { e.sugar.Errorf(format, args...) }
func (e *SugaredEntry) Debugf(format string, args ...any)         { e.sugar.Debugf(format, args...) }
func (e *SugaredEntry) Fatalf(format string, args ...any)         { e.sugar.Fatalf(format, args...) }
func (e *SugaredEntry) Panicf(format string, args ...any)         { e.sugar.Panicf(format, args...) }

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