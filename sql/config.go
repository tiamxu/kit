package sql

import (
	"fmt"
	"strings"

	// 注册常用驱动
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type Config struct {
	Driver          string `yaml:"driver" json:"driver"`
	Database        string `yaml:"database" json:"database"`
	Username        string `yaml:"username" json:"username"`
	Password        string `yaml:"password" json:"-"`
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	MaxIdleConns    int    `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxOpenConns    int    `yaml:"max_open_conns" json:"max_open_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
	ReadTimeout     int    `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    int    `yaml:"write_timeout" json:"write_timeout"`
}

func (cfg *Config) Validate() error {
	if cfg.Driver == "" {
		return fmt.Errorf("sql: driver is required")
	}
	supportedDrivers := map[string]bool{
		"mysql": true, "postgres": true, "clickhouse": true,
	}
	if !supportedDrivers[strings.ToLower(cfg.Driver)] {
		return fmt.Errorf("sql: unsupported driver %q, supported: mysql, postgres, clickhouse", cfg.Driver)
	}
	if cfg.Host == "" {
		return fmt.Errorf("sql: host is required")
	}
	if cfg.Database == "" {
		return fmt.Errorf("sql: database is required")
	}
	return nil
}

func (cfg *Config) Source() (string, error) {
	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		return cfg.mysqlSource(), nil
	case "postgres":
		return cfg.postgresSource(), nil
	case "clickhouse":
		return cfg.clickHouseSource(), nil
	default:
		return "", fmt.Errorf("sql: unsupported driver %q", cfg.Driver)
	}
}

func (cfg *Config) mysqlSource() string {
	pwd := cfg.Password
	if pwd != "" {
		pwd = ":" + pwd
	}
	username := cfg.Username
	if username == "" {
		username = "root"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	return fmt.Sprintf("%s%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
		username, pwd, cfg.Host, port, cfg.Database)
}

func (cfg *Config) postgresSource() string {
	pwd := cfg.Password
	if pwd != "" {
		pwd = ":" + pwd
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username, pwd, cfg.Host, port, cfg.Database)
}

func (cfg *Config) clickHouseSource() string {
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 10
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10
	}
	if cfg.Port == 0 {
		cfg.Port = 9000
	}
	return fmt.Sprintf("clickhouse://%s:%d?username=%s&database=%s&read_timeout=%d&write_timeout=%d",
		cfg.Host, cfg.Port, cfg.Username, cfg.Database, cfg.ReadTimeout, cfg.WriteTimeout)
}
