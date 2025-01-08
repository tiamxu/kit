package log

import (
	"context"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type kafkaWriter struct {
	writer *kafka.Writer
	topic  string
}

func newKafkaWriter(brokers []string, topic string) *kafkaWriter {
	return &kafkaWriter{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 100 * time.Millisecond,
		},
		topic: topic,
	}
}

func (w *kafkaWriter) Write(p []byte) (int, error) {
	msg := kafka.Message{
		Key:   []byte(time.Now().Format(time.RFC3339)),
		Value: p,
	}

	err := w.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *kafkaWriter) Close() error {
	return w.writer.Close()
}

func setupKafkaOutput(cfg *Config) (logrus.Formatter, io.Writer) {
	// TODO: Make brokers configurable
	brokers := []string{"localhost:9092"}
	writer := newKafkaWriter(brokers, cfg.Topic)

	formatter := &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}

	return formatter, writer
}
