package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer 封装了使用segmentio/kafka-go的Kafka生产者
type KafkaProducer struct {
	writer *kafka.Writer
	config KafkaConfig
}

type KafkaConfig struct {
	Brokers       []string
	Topic         string
	MaxRetries    int           // 最大重试次数
	RetryInterval time.Duration // 重试间隔
	BatchTimeout  time.Duration // 批量提交超时
	BatchSize     int           // 批量大小
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(cfg KafkaConfig) (*KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("brokers cannot be empty")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	// 设置默认值
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

	// 创建Kafka writer配置
	config := kafka.WriterConfig{
		Brokers:      cfg.Brokers,
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: cfg.BatchTimeout,
		BatchSize:    cfg.BatchSize,
	}

	// 创建Kafka writer实例
	writer := kafka.NewWriter(config)

	p := &KafkaProducer{
		writer: writer,
		config: cfg,
	}
	return p, nil
}

// SendMessage 发送消息到Kafka
func (p *KafkaProducer) SendMessage(key, value []byte) error {
	// 创建消息
	message := kafka.Message{
		Key:   key,
		Value: value,
	}

	// 发送消息并处理可能的错误
	err := p.writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}

	return nil
}

// KafkaConsumer 封装了使用segmentio/kafka-go的Kafka消费者
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer 创建一个新的Kafka消费者
func NewKafkaConsumer(brokers []string, topic string, groupID string) (*KafkaConsumer, error) {
	// 创建Kafka reader配置
	config := kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6, //10MB
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second, // 每秒刷新一次提交给 Kafka
	}

	// 创建Kafka reader实例
	reader := kafka.NewReader(config)

	c := &KafkaConsumer{
		reader: reader,
	}
	return c, nil
}

// ConsumeMessage 从Kafka消费消息
func (c *KafkaConsumer) ConsumeMessage() {
	for {
		// 读取消息并处理可能的错误
		message, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("found error from kafka reader %v", err)
			continue
		}

		// 打印接收到的消息内容
		fmt.Printf("Received message: Key: %s, Value: %s\n", string(message.Key), string(message.Value))
	}
}
