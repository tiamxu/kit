package sql

import (
	"fmt"
	"strings"
)

type Config struct {
	Driver          string `yaml:"driver"`
	Database        string `yaml:"database"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	// ReadTimeout clickhouse read timeout in second
	ReadTimeout int `yaml:"read_timeout"`
	// WriteTimeout clickhouse write timeout in second
	WriteTimeout int `yaml:"write_timeout"`
}

func (cfg *Config) Source() string {
	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		return cfg.mysqlSource()
	case "postgres":
		return cfg.postgresSource()
	case "clickhouse":
		return cfg.clickHouseSource()
	default:
		return ""

	}
}

func (cfg *Config) mysqlSource() string {
	pwd := cfg.Password
	if pwd != "" {
		pwd = ":" + pwd
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	dbSource := fmt.Sprintf("%s%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true", cfg.Username, pwd, cfg.Host, port, cfg.Database)
	return dbSource
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
	dbSource := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.Username, pwd, cfg.Host, port, cfg.Database)
	return dbSource
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
	//clickhouse://username:password@host:port/database?debug=false
	dbSource := fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s&read_timeout=%d&write_timeout=%d", cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.ReadTimeout, cfg.WriteTimeout)
	return dbSource
}
