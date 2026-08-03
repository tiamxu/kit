package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLoggerBundleInfoRecordJSONOmitsMessage(t *testing.T) {
	var output bytes.Buffer
	bundle, err := newLoggerBundle(&Config{Level: "info", Format: "json"}, []zapcore.WriteSyncer{zapcore.AddSync(&output)})
	if err != nil {
		t.Fatalf("newLoggerBundle returned error: %v", err)
	}

	bundle.infoRecord(Fields{"status": 200, "method": "GET"}, "200 GET")

	var record map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if _, ok := record["msg"]; ok {
		t.Fatalf("record contains msg: %#v", record)
	}
	if record["status"] != float64(200) || record["method"] != "GET" {
		t.Fatalf("record fields = %#v", record)
	}
}

func TestLoggerBundleInfoRecordConsoleUsesProvidedText(t *testing.T) {
	var output bytes.Buffer
	bundle, err := newLoggerBundle(&Config{Level: "info", Format: "console"}, []zapcore.WriteSyncer{zapcore.AddSync(&output)})
	if err != nil {
		t.Fatalf("newLoggerBundle returned error: %v", err)
	}

	bundle.infoRecord(Fields{"status": 200, "method": "GET"}, "200 GET /api/v1/teams")

	text := output.String()
	if !strings.Contains(text, "200 GET /api/v1/teams") {
		t.Fatalf("console output = %q", text)
	}
	if strings.Contains(text, `{"status"`) || strings.Contains(text, "access") {
		t.Fatalf("console output contains structured suffix: %q", text)
	}
}

func TestLoggerBundleNormalJSONKeepsMessage(t *testing.T) {
	var output bytes.Buffer
	bundle, err := newLoggerBundle(&Config{Level: "info", Format: "json"}, []zapcore.WriteSyncer{zapcore.AddSync(&output)})
	if err != nil {
		t.Fatalf("newLoggerBundle returned error: %v", err)
	}

	bundle.sugar.With("module", "auth").Info("login succeeded")

	var record map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if record["msg"] != "login succeeded" || record["module"] != "auth" {
		t.Fatalf("record = %#v", record)
	}
}
