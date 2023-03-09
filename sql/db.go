package sql

import (
	_ "database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

// Connect to a database and verify with a ping.
func Connect(dbConfig *Config) (err error) {
	db, err = sqlx.Connect(dbConfig.Driver, dbConfig.Source())
	if err != nil {
		return
	}
	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetime) * time.Second)
	return
}
