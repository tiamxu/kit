package http

import (
	"context"
	"net"
	"testing"
	"time"

	kiterrors "github.com/tiamxu/kit/errors"
)

func TestNewServerDoesNotListen(t *testing.T) {
	addr := unusedTCPAddr(t)

	srv := NewServer(ServerConfig{Address: addr})
	if srv == nil {
		t.Fatal("expected server")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("expected address to remain available before Start: %v", err)
	}
	_ = listener.Close()
}

func TestServerStartListens(t *testing.T) {
	addr := unusedTCPAddr(t)
	srv := NewServer(ServerConfig{Address: addr})

	if err := srv.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer srv.Shutdown(context.Background())

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("expected server to listen after Start: %v", err)
	}
	_ = conn.Close()
}

func TestServerStartReturnsHTTPStartWhenAddressInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen occupied address: %v", err)
	}
	defer listener.Close()

	srv := NewServer(ServerConfig{Address: listener.Addr().String()})

	err = srv.Start()
	if err == nil {
		t.Fatal("expected Start error")
	}
	if code := kiterrors.GetCode(err); code != "HTTP_START" {
		t.Fatalf("expected HTTP_START, got %q: %v", code, err)
	}
}

func unusedTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen unused tcp addr: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close unused tcp listener: %v", err)
	}
	return addr
}
