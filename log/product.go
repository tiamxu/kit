package log

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tiamxu/kit/kafka"
)

var kafkaProducer *kafka.KafkaProducer

func setupKafkaOutput(cfg *Config) (logrus.Formatter, *kafkaWriter) {
	// 初始化kafka producer
	kafkaCfg := kafka.Config{
		Brokers:    cfg.KafkaConfig.Brokers,
		Topic:      cfg.KafkaConfig.Topic,
		MaxRetries: cfg.KafkaConfig.MaxRetries,
	}

	var err error
	kafkaProducer, err = kafka.NewKafkaProducer(&kafkaCfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to init kafka producer: %v", err))
	}

	// 创建kafka writer
	kw := &kafkaWriter{
		producer: kafkaProducer,
		topic:    cfg.KafkaConfig.Topic,
	}

	return &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}, kw
}

type kafkaWriter struct {
	producer *kafka.KafkaProducer
	topic    string
}

func (w *kafkaWriter) Write(p []byte) (n int, err error) {
	err = w.producer.SendMessage(nil, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
