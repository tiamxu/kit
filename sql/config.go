package sql

import (
	"fmt"
	"net/url"
	"strings"

	// 注册常用驱动
	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
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
		pwd = ":" + url.QueryEscape(pwd)
	}
	username := cfg.Username
	if username == "" {
		username = "root"
	}
	username = url.QueryEscape(username)
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	return fmt.Sprintf("%s%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
		username, pwd, cfg.Host, port, cfg.Database)
}

func (cfg *Config) postgresSource() string {
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	u := &url.URL{
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%d", cfg.Host, port),
		Path:     "/" + cfg.Database,
		RawQuery: url.Values{"sslmode": []string{"disable"}}.Encode(),
	}
	if cfg.Password != "" {
		u.User = url.UserPassword(cfg.Username, cfg.Password)
	} else if cfg.Username != "" {
		u.User = url.User(cfg.Username)
	}
	return u.String()
}

func (cfg *Config) clickHouseSource() string {
	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 10
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 10
	}
	port := cfg.Port
	if port == 0 {
		port = 9000
	}
	values := url.Values{}
	values.Set("username", cfg.Username)
	if cfg.Password != "" {
		values.Set("password", cfg.Password)
	}
	values.Set("database", cfg.Database)
	values.Set("read_timeout", fmt.Sprintf("%d", readTimeout))
	values.Set("write_timeout", fmt.Sprintf("%d", writeTimeout))
	return fmt.Sprintf("clickhouse://%s:%d?%s", cfg.Host, port, values.Encode())
}
