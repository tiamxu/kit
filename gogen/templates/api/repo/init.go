package repo

import (
	"github.com/tiamxu/kit/sql"
)

var DB *sql.DB

func Init(cfg *sql.Config) error {
	db, err := sql.Connect(cfg)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func IsNoRows(err error) bool {
	return sql.IsNoRows(err)
}
