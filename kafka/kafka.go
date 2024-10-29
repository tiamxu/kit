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
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(brokers []string, topic string) (*KafkaProducer, error) {
	// 创建Kafka writer配置
	config := kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// 创建Kafka writer实例
	writer := kafka.NewWriter(config)

	p := &KafkaProducer{
		writer: writer,
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
