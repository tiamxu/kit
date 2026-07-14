package redis

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestTryLockStopsDuringRetryDelayWhenContextCancelled(t *testing.T) {
	addr, closeServer := startSetNXFalseServer(t)
	defer closeServer()

	client := &Client{Client: goredis.NewClient(&goredis.Options{Addr: addr})}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.TryLock(ctx, "resource", time.Minute, WithLockRetry(2, time.Hour))
		result <- err
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
		t.Fatal("TryLock did not return after context cancellation")
	}
}

func startSetNXFalseServer(t *testing.T) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake redis: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "*") {
				if _, err := conn.Write([]byte("$-1\r\n")); err != nil {
					return
				}
			}
		}
	}()

	closeFn := func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
	return listener.Addr().String(), closeFn
}
