package sql

import (
	"strings"
	"testing"
)

func TestMySQLSourceEscapesUserInfo(t *testing.T) {
	cfg := &Config{
		Driver:   "mysql",
		Host:     "localhost",
		Database: "app",
		Username: "user@name",
		Password: "p@ss:word",
	}

	dsn, err := cfg.Source()
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}

	if !strings.HasPrefix(dsn, "user%40name:p%40ss%3Aword@tcp(localhost:3306)/app?") {
		t.Fatalf("mysql dsn did not escape user info: %s", dsn)
	}
}

func TestClickHouseSourceIncludesEscapedPassword(t *testing.T) {
	cfg := &Config{
		Driver:   "clickhouse",
		Host:     "localhost",
		Database: "analytics",
		Username: "user@name",
		Password: "p@ss word",
	}

	dsn, err := cfg.Source()
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}

	for _, want := range []string{
		"username=user%40name",
		"password=p%40ss+word",
		"database=analytics",
		"read_timeout=10",
		"write_timeout=10",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("clickhouse dsn %q missing %q", dsn, want)
		}
	}
}
