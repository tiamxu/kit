package v1

import (
	"context"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// KafkaProducer 封装了Kafka生产者
type KafkaProducer struct {
	producer sarama.AsyncProducer
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	// 发送完消息后需要 leader 和 follower 都确认
	config.Producer.RequiredAcks = sarama.WaitForLocal
	// 使用 Snappy 压缩
	config.Producer.Compression = sarama.CompressionSnappy
	// 每 500ms 刷新一次消息缓冲
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Return.Successes = true
	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	p := &KafkaProducer{
		producer: producer,
	}
	return p, nil
}

// SendMessage 发送消息到Kafka
func (p *KafkaProducer) SendMessage(topic string, key, value []byte) {
	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	p.producer.Input() <- message
}

// KafkaConsumer 封装了Kafka消费者
type KafkaConsumer struct {
	topic     string
	partition int32
	consumer  sarama.ConsumerGroup
}

// NewKafkaConsumer 创建一个新的Kafka消费者
func NewKafkaConsumer(brokers []string, group, topic string, partition int32) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	consumerGroup, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		return nil, err
	}
	c := &KafkaConsumer{
		topic:     topic,
		consumer:  consumerGroup,
		partition: partition,
	}
	return c, nil
}

// ConsumeMessage 从Kafka消费消息
func (c *KafkaConsumer) ConsumeMessage(handler sarama.ConsumerGroupHandler) {
	defer c.consumer.Close()
	for {
		if err := c.consumer.Consume(context.Background(), []string{c.topic}, handler); err != nil {
			log.Printf("found error from kafka group consumer %v", err)
		}
	}
}
