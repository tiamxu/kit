package kafka

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestBuildWriterConfigDoesNotBindTopic(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "events",
	}

	writerCfg := buildWriterConfig(cfg)

	if writerCfg.Topic != "" {
		t.Fatalf("expected writer topic to be empty, got %q", writerCfg.Topic)
	}
}

func TestKafkaProducerResolveTopicUsesConfiguredDefault(t *testing.T) {
	producer := &KafkaProducer{config: &Config{Topic: "events"}}

	if got := producer.resolveTopic(""); got != "events" {
		t.Fatalf("expected default topic, got %q", got)
	}
	if got := producer.resolveTopic("audit"); got != "audit" {
		t.Fatalf("expected explicit topic, got %q", got)
	}
}

func TestSendMessageCtxStopsDuringRetryDelayWhenContextCancelled(t *testing.T) {
	addr := unusedTCPAddr(t)
	producer, err := NewKafkaProducer(&Config{
		Brokers:       []string{addr},
		Topic:         "events",
		MaxRetries:    2,
		RetryInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewKafkaProducer returned error: %v", err)
	}
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- producer.SendMessageCtx(ctx, "", []byte("k"), []byte("v"))
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected wrapped context.Canceled, got %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("SendMessageCtx did not return after context cancellation")
	}
}

func unusedTCPAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen unused tcp addr: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close unused tcp listener: %v", err)
	}
	return addr
}
