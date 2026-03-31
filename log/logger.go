package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/tiamxu/kit/kafka"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultMaxSize           = 100
	defaultMaxBackups        = 5
	defaultMaxAge            = 30
	defaultKafkaBatchSize    = 10000
	defaultKafkaBatchTimeout = 5 * time.Second
	defaultKafkaChanSize     = 10000
)

type Config struct {
	Level      string       `yaml:"level" json:"level"`
	FilePath   string       `yaml:"file_path" json:"file_path"`
	FileName   string       `yaml:"file_name" json:"file_name"`
	MaxSize    int          `yaml:"max_size" json:"max_size"`
	MaxBackups int          `yaml:"max_backups" json:"max_backups"`
	MaxAge     int          `yaml:"max_age" json:"max_age"`
	Compress   bool         `yaml:"compress" json:"compress"`
	Type       string       `yaml:"type" json:"type"`
	Format     string       `yaml:"format" json:"format"`
	Kafka      kafka.Config `yaml:"kafka" json:"kafka"`
}

type Fields = map[string]interface{}

var (
	_sugar       *zap.SugaredLogger
	_logger      *zap.Logger
	_mu          sync.RWMutex
	_fields      Fields
	_kafkaWriter *kafkaLogWriter
)

func InitLogger(cfg *Config) error {
	_mu.Lock()
	defer _mu.Unlock()

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	outputType := strings.ToLower(cfg.Type)
	format := strings.ToLower(cfg.Format)

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
		case "kafka":
			kafkaCore, writer, err := buildKafkaCore(cfg, enabler)
			if err != nil {
				return fmt.Errorf("setup kafka output: %w", err)
			}
			_kafkaWriter = writer
			cores = append(cores, kafkaCore)
		default:
			return fmt.Errorf("unknown output type: %s", part)
		}
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), enabler))
	}

	core := zapcore.NewTee(cores...)
	_logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	_sugar = _logger.Sugar()

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

func buildKafkaCore(cfg *Config, enabler zapcore.LevelEnabler) (zapcore.Core, *kafkaLogWriter, error) {
	kafkaCfg := cfg.Kafka
	if len(kafkaCfg.Brokers) == 0 {
		return nil, nil, fmt.Errorf("kafka brokers is required")
	}
	if kafkaCfg.Topic == "" {
		return nil, nil, fmt.Errorf("kafka topic is required")
	}
	if kafkaCfg.BatchSize <= 0 {
		kafkaCfg.BatchSize = defaultKafkaBatchSize
	}
	if kafkaCfg.BatchTimeout <= 0 {
		kafkaCfg.BatchTimeout = defaultKafkaBatchTimeout
	}

	producer, err := kafka.NewKafkaProducer(&kafkaCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create kafka producer: %w", err)
	}

	writer := &kafkaLogWriter{
		producer: producer,
		topic:    kafkaCfg.Topic,
		ch:       make(chan []byte, defaultKafkaChanSize),
		done:     make(chan struct{}),
	}
	go writer.run()

	// Kafka 端强制使用 JSON 格式，方便下游消费
	kafkaEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
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
	})

	core := zapcore.NewCore(kafkaEncoder, zapcore.AddSync(writer), enabler)
	return core, writer, nil
}

type kafkaLogWriter struct {
	producer *kafka.KafkaProducer
	topic    string
	ch       chan []byte
	done     chan struct{}
}

func (w *kafkaLogWriter) run() {
	for {
		select {
		case msg := <-w.ch:
			_ = w.producer.SendMessage(w.topic, nil, msg)
		case <-w.done:
			for {
				select {
				case msg := <-w.ch:
					_ = w.producer.SendMessage(w.topic, nil, msg)
				default:
					return
				}
			}
		}
	}
}

func (w *kafkaLogWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	cp := make([]byte, len(p))
	copy(cp, p)
	select {
	case w.ch <- cp:
		return len(p), nil
	default:
		return len(p), nil
	}
}

func (w *kafkaLogWriter) Close() error {
	close(w.done)
	for {
		select {
		case <-w.ch:
		default:
			close(w.ch)
			return w.producer.Close()
		}
	}
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

func ensureLogger() {
	_mu.RLock()
	if _sugar != nil {
		_mu.RUnlock()
		return
	}
	_mu.RUnlock()

	_mu.Lock()
	defer _mu.Unlock()
	if _sugar != nil {
		return
	}
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:      "time",
		LevelKey:     "level",
		NameKey:      "logger",
		CallerKey:    "caller",
		MessageKey:   "msg",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.InfoLevel)
	_logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	_sugar = _logger.Sugar()
}

func GetLogger() *zap.SugaredLogger {
	ensureLogger()
	return _sugar
}

func SetGlobalFields(fields Fields) {
	_mu.Lock()
	defer _mu.Unlock()
	_fields = fields
}

func getMergedFields(fields Fields) []interface{} {
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

func fieldsToArgs(fields Fields) []interface{} {
	if len(fields) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(fields)*2)
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
	return &SugaredEntry{sugar: _sugar.With(args...)}
}

func WithContext(ctx context.Context) *SugaredEntry {
	ensureLogger()
	args := []interface{}{}
	if requestID := ctx.Value("request_id"); requestID != nil {
		args = append(args, "request_id", requestID)
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		args = append(args, "trace_id", traceID)
	}
	return &SugaredEntry{sugar: _sugar.With(args...)}
}

func (e *SugaredEntry) Info(msg string)                           { e.sugar.Infow(msg) }
func (e *SugaredEntry) Warn(msg string)                           { e.sugar.Warnw(msg) }
func (e *SugaredEntry) Error(msg string)                          { e.sugar.Errorw(msg) }
func (e *SugaredEntry) Debug(msg string)                          { e.sugar.Debugw(msg) }
func (e *SugaredEntry) Fatal(msg string)                          { e.sugar.Fatalw(msg) }
func (e *SugaredEntry) Panic(msg string)                          { e.sugar.Panicw(msg) }
func (e *SugaredEntry) Infof(format string, args ...interface{})  { e.sugar.Infof(format, args...) }
func (e *SugaredEntry) Warnf(format string, args ...interface{})  { e.sugar.Warnf(format, args...) }
func (e *SugaredEntry) Errorf(format string, args ...interface{}) { e.sugar.Errorf(format, args...) }
func (e *SugaredEntry) Debugf(format string, args ...interface{}) { e.sugar.Debugf(format, args...) }
func (e *SugaredEntry) Fatalf(format string, args ...interface{}) { e.sugar.Fatalf(format, args...) }
func (e *SugaredEntry) Panicf(format string, args ...interface{}) { e.sugar.Panicf(format, args...) }

func Tracef(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Debugf(format, args...)
}

func Traceln(args ...interface{}) {
	ensureLogger()
	_sugar.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Debugf(format, args...)
}

func Debugln(args ...interface{}) {
	ensureLogger()
	_sugar.Debug(args...)
}

func Printf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Infof(format, args...)
}

func Println(args ...interface{}) {
	ensureLogger()
	_sugar.Info(args...)
}

func Infof(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Infof(format, args...)
}

func Infoln(args ...interface{}) {
	ensureLogger()
	_sugar.Info(args...)
}

func Warnf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Warnf(format, args...)
}

func Warnln(args ...interface{}) {
	ensureLogger()
	_sugar.Warn(args...)
}

func Errorf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Errorf(format, args...)
}

func Errorln(args ...interface{}) {
	ensureLogger()
	_sugar.Error(args...)
}

func Panicf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Panicf(format, args...)
}

func Panicln(args ...interface{}) {
	ensureLogger()
	_sugar.Panic(args...)
}

func Fatalf(format string, args ...interface{}) {
	ensureLogger()
	_sugar.Fatalf(format, args...)
}

func Fatalln(args ...interface{}) {
	ensureLogger()
	_sugar.Fatal(args...)
}

func Sync() {
	_mu.Lock()
	defer _mu.Unlock()
	if _kafkaWriter != nil {
		_ = _kafkaWriter.Close()
		_kafkaWriter = nil
	}
	if _logger != nil {
		_ = _logger.Sync()
	}
}

var _ = time.Now()
