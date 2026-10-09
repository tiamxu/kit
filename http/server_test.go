package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	kiterrors "github.com/tiamxu/kit/errors"
	"github.com/tiamxu/kit/log"
)

func TestFormatAccessLogConsoleUsesValuesInJSONFieldOrder(t *testing.T) {
	fields := log.Fields{
		"status": 200, "method": "GET", "path": "/api/v1/teams",
		"query": "page=1&page_size=20", "client_ip": "::1", "host": "localhost:8800",
		"request_id": "request-id", "user_agent": "Mozilla Test", "request_time": "0.627s",
		"bytes_out": 2413, "referer": "",
	}
	want := `200 GET /api/v1/teams page=1&page_size=20 ::1 localhost:8800 request-id "Mozilla Test" 0.627s 2413 -`
	if got := formatAccessLogConsole(fields); got != want {
		t.Fatalf("formatAccessLogConsole() = %q, want %q", got, want)
	}
}

func TestFormatAccessLogValueQuotesControlCharacters(t *testing.T) {
	want := `"/api/v1/teams\x00\x1b"`
	if got := formatAccessLogValue("/api/v1/teams\x00\x1b"); got != want {
		t.Fatalf("formatAccessLogValue() = %q, want %q", got, want)
	}
}

func TestAccessLogMiddlewareRecordsHTTPAndHijackedResponses(t *testing.T) {
	logDir := t.TempDir()
	if err := log.InitLogger(&log.Config{
		Level:    "info",
		Type:     "file",
		Format:   "json",
		FilePath: logDir,
		FileName: "access.log",
	}); err != nil {
		t.Fatalf("init logger: %v", err)
	}

	router := NewGin(ServerConfig{})
	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/error", func(c *gin.Context) {
		_ = c.Error(errors.New("validation failed")).SetType(gin.ErrorTypePublic)
	})
	router.GET("/panic", func(_ *gin.Context) {
		panic("boom")
	})
	router.GET("/hijack-unknown", func(c *gin.Context) {
		conn, _, err := c.Writer.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		conn.Close()
	})
	router.GET("/upgrade-rejected", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusForbidden)
	})
	for _, scenario := range []struct {
		path     string
		buffered bool
		status   int
	}{
		{"/upgrade-direct", false, http.StatusSwitchingProtocols},
		{"/upgrade-buffered", true, http.StatusSwitchingProtocols},
		{"/hijack-rejected", false, http.StatusForbidden},
	} {
		router.GET(scenario.path, func(c *gin.Context) {
			conn, rw, err := c.Writer.Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			handshake := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Length: 2\r\n", scenario.status, http.StatusText(scenario.status))
			if scenario.status == http.StatusSwitchingProtocols {
				handshake += "Connection: Upgrade\r\nUpgrade: websocket\r\n"
			}
			handshake += "\r\nOK"
			if scenario.buffered {
				_, err = rw.WriteString(handshake)
				if err == nil {
					err = rw.Flush()
				}
			} else {
				_, err = conn.Write([]byte(handshake[:8]))
				if err == nil {
					_, err = conn.Write([]byte(handshake[8:]))
				}
			}
			if err != nil {
				t.Error(err)
			}
		})
	}

	expectedStatuses := map[string]int{
		"/ok":    http.StatusOK,
		"/error": http.StatusUnprocessableEntity,
		"/panic": http.StatusInternalServerError,
	}
	requestIDs := make(map[string]string, len(expectedStatuses))
	for path, expectedStatus := range expectedStatuses {
		requestID := "request-id-" + strings.TrimPrefix(path, "/")
		requestIDs[path] = requestID
		req := httptest.NewRequest(http.MethodGet, path+"?page=1", nil)
		req.Header.Set("X-Request-ID", requestID)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != expectedStatus {
			t.Fatalf("%s status = %d, want %d", path, resp.Code, expectedStatus)
		}
	}
	finished := make(chan struct{}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
		finished <- struct{}{}
	}))
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	for path, expectedStatus := range map[string]int{
		"/upgrade-direct":   http.StatusSwitchingProtocols,
		"/upgrade-buffered": http.StatusSwitchingProtocols,
		"/upgrade-rejected": http.StatusForbidden,
		"/hijack-rejected":  http.StatusForbidden,
	} {
		req, err := http.NewRequest(http.MethodGet, server.URL+path+"?page=1", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		requestIDs[path] = "request-id-" + strings.TrimPrefix(path, "/")
		req.Header.Set("X-Request-ID", requestIDs[path])
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if path != "/upgrade-rejected" && string(body) != "OK" {
			t.Fatalf("%s body = %q, want OK", path, body)
		}
		if resp.StatusCode != expectedStatus {
			t.Fatalf("upgrade status=%d", resp.StatusCode)
		}
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Fatal("access middleware did not finish")
		}
		expectedStatuses[path] = expectedStatus
	}

	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprint(conn, "GET /hijack-unknown?page=1 HTTP/1.1\r\nHost: localhost\r\nX-Request-ID: request-id-unknown\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := io.ReadAll(conn); err != nil || len(body) != 0 {
		t.Fatalf("unknown hijack response = %q, err = %v", body, err)
	}
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("unknown hijack middleware did not finish")
	}
	expectedStatuses["/hijack-unknown"] = 0
	requestIDs["/hijack-unknown"] = "request-id-unknown"

	if err := log.Sync(); err != nil {
		t.Fatalf("sync logger: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(logDir, "access.log"))
	if err != nil {
		t.Fatalf("read access log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(expectedStatuses)+1 {
		t.Fatalf("log line count = %d, want %d: %s", len(lines), len(expectedStatuses)+1, data)
	}

	infoRecords := make(map[string]map[string]any, 3)
	errorCount := 0
	for _, line := range lines {
		if strings.Count(line, `"time":`) != 1 {
			t.Fatalf("log must contain one time field: %s", line)
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log record: %v: %s", err, line)
		}
		switch record["level"] {
		case "info":
			path, _ := record["path"].(string)
			infoRecords[path] = record
			for _, field := range []string{"msg", "error", "error_count", "bytes_in", "real_ip", "protocol"} {
				if _, ok := record[field]; ok {
					t.Fatalf("access log contains %s: %#v", field, record)
				}
			}
			caller, _ := record["caller"].(string)
			if !strings.Contains(caller, "http/server.go:") {
				t.Fatalf("access log caller = %q", caller)
			}
		case "error":
			errorCount++
			if record["msg"] != "validation failed" {
				t.Fatalf("error log msg = %#v", record["msg"])
			}
		default:
			t.Fatalf("unexpected log level: %#v", record)
		}
	}

	if errorCount != 1 {
		t.Fatalf("error log count = %d, want 1", errorCount)
	}
	for path, expectedStatus := range expectedStatuses {
		record, ok := infoRecords[path]
		if !ok {
			t.Fatalf("missing access log for %s: %s", path, data)
		}
		var wantStatus any
		if expectedStatus != 0 {
			wantStatus = float64(expectedStatus)
		}
		if record["status"] != wantStatus {
			t.Errorf("%s access status = %#v, want %d", path, record["status"], expectedStatus)
		}
		if record["request_id"] != requestIDs[path] {
			t.Fatalf("%s request_id = %#v, want %q", path, record["request_id"], requestIDs[path])
		}
		if record["query"] != "page=1" {
			t.Fatalf("%s query = %#v", path, record["query"])
		}
	}
}

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
