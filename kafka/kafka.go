package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	kiterrors "github.com/tiamxu/kit/errors"
	"github.com/tiamxu/kit/log"
)

// KafkaProducer 封装了使用segmentio/kafka-go的Kafka生产者
type KafkaProducer struct {
	writer *kafka.Writer
	config *Config
}

// Config Kafka生产者配置
type Config struct {
	Brokers       []string      `yaml:"brokers"`        // Kafka broker地址列表
	Topic         string        `yaml:"topic"`          // 默认主题
	MaxRetries    int           `yaml:"max_retries"`    // 最大重试次数
	RetryInterval time.Duration `yaml:"retry_interval"` // 重试间隔
	BatchTimeout  time.Duration `yaml:"batch_timeout"`  // 批量提交超时
	BatchSize     int           `yaml:"batch_size"`     // 批量大小
}

// NewKafkaProducer 创建一个新的Kafka生产者
// 参数:
//
//	cfg: Kafka生产者配置
//
// 返回:
//
//	*KafkaProducer: Kafka生产者实例
//	error: 错误信息
func NewKafkaProducer(cfg *Config) (*KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, kiterrors.Wrap("KAFKA_PARAM", "brokers cannot be empty", kiterrors.KafkaErrNoBrokers)
	}
	if cfg.Topic == "" {
		return nil, kiterrors.Wrap("KAFKA_PARAM", "topic cannot be empty", kiterrors.KafkaErrTopic)
	}

	applyDefaults(cfg)

	writerConfig := buildWriterConfig(cfg)
	writer := kafka.NewWriter(writerConfig)
	writer.AllowAutoTopicCreation = true

	return &KafkaProducer{
		writer: writer,
		config: cfg,
	}, nil
}

func applyDefaults(cfg *Config) {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = 100 * time.Millisecond
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
}

func buildWriterConfig(cfg *Config) kafka.WriterConfig {
	return kafka.WriterConfig{
		Brokers:      cfg.Brokers,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: cfg.BatchTimeout,
		BatchSize:    cfg.BatchSize,
		Async:        true,
	}
}

// Close 关闭Kafka生产者
// 返回:
//
//	error: 错误信息
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// SendMessage 发送消息到Kafka（带超时控制）
// 参数:
//
//	topic: 主题
//	key: 消息键
//	value: 消息值
//
// 返回:
//
//	error: 错误信息
func (p *KafkaProducer) SendMessage(topic string, key, value []byte) error {
	return p.SendMessageCtx(context.Background(), topic, key, value)
}

func (p *KafkaProducer) SendMessageCtx(ctx context.Context, topic string, key, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var lastErr error
	for i := 0; i < p.config.MaxRetries; i++ {
		err := p.writer.WriteMessages(ctx, msg)
		if err == nil {
			return nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return kiterrors.Wrap("KAFKA_PRODUCE", "context cancelled", err)
		}
		if i < p.config.MaxRetries-1 {
			time.Sleep(p.config.RetryInterval)
		}
	}
	return kiterrors.Wrap("KAFKA_PRODUCE", fmt.Sprintf("failed to send message after %d retries", p.config.MaxRetries), lastErr)
}

// KafkaConsumer 封装了使用segmentio/kafka-go的Kafka消费者
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer 创建一个新的Kafka消费者
// 参数:
//
//	brokers: Kafka broker地址列表
//	topic: 主题
//	groupID: 消费者组ID
//
// 返回:
//
//	*KafkaConsumer: Kafka消费者实例
//	error: 错误信息
func NewKafkaConsumer(brokers []string, topic string, groupID string) (*KafkaConsumer, error) {
	if len(brokers) == 0 {
		return nil, kiterrors.Wrap("KAFKA_PARAM", "brokers cannot be empty", kiterrors.KafkaErrNoBrokers)
	}
	if topic == "" {
		return nil, kiterrors.Wrap("KAFKA_PARAM", "topic cannot be empty", kiterrors.KafkaErrTopic)
	}
	if groupID == "" {
		return nil, kiterrors.Wrap("KAFKA_PARAM", "groupID cannot be empty", kiterrors.ErrInvalidParam)
	}
	// 创建Kafka reader配置
	readerConfig := kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6, // 10MB
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second, // 每秒刷新一次提交给 Kafka
	}
	// 创建Kafka reader实例
	reader := kafka.NewReader(readerConfig)
	return &KafkaConsumer{
		reader: reader,
	}, nil
}

// ConsumeMessage 从Kafka消费消息，返回消息channel
// 调用方通过 channel 接收消息，处理完后调用 Ack
func (c *KafkaConsumer) ConsumeMessage(ctx context.Context) (<-chan kafka.Message, error) {
	msgChan := make(chan kafka.Message, 100)
	go func() {
		defer close(msgChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Errorf("kafka fetch message error: %v", err)
				continue
			}
			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return msgChan, nil
}

// Ack 确认消息已处理
func (c *KafkaConsumer) Ack(ctx context.Context, msg kafka.Message) error {
	return c.reader.CommitMessages(ctx, msg)
}

// Close 关闭消费者
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
