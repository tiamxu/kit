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
}

func (cfg *Config) Source() string {
	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		return cfg.mysqlSource()
	case "postgres":
		return cfg.postgresSource()
	default:
		return ""

	}
}

func (cfg *Config) mysqlSource() string {

	dbSource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return dbSource
}
func (cfg *Config) postgresSource() string {
	dbSource := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return dbSource
}
