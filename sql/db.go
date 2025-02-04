package sql

import (
	"database/sql"
	"time"

	_ "github.com/ClickHouse/clickhouse-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	*sqlx.DB
	dbConfig *Config
}

//var db *sqlx.DB

// Connect to a database and verify with a ping.
func Connect(dbConfig *Config) (*DB, error) {
	db, err := sqlx.Connect(dbConfig.Driver, dbConfig.Source())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetime) * time.Second)

	return &DB{
		DB:       db,
		dbConfig: dbConfig,
	}, nil
}

// Callback non-transactional operations.
func (d *DB) Callback(fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) error {
	if fn == nil {
		return nil
	}
	if len(tx) > 0 && tx[0] != nil {
		return fn(tx[0])
	}
	return fn(nil)
}

// TransactCallback transactional operations.
// nOTE: if an error is returned, the rollback method should be invoked outside the function.
func (d *DB) TransactCallback(fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) (err error) {
	if fn == nil {
		return
	}

	var _tx *sqlx.Tx
	if len(tx) > 0 {
		_tx = tx[0]
	}
	if _tx == nil {
		_tx, err = d.Beginx()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_tx.Rollback()
			} else {
				_tx.Commit()
			}
		}()
	}
	err = fn(_tx)
	return err
}

var ErrNoRows = sql.ErrNoRows

// IsNoRows is the data exist or not.
func IsNoRows(err error) bool {
	return ErrNoRows == err
}

// PreDB preset *DB
type PreDB struct {
	*DB
	inited bool
}

// NewPreDB creates a unconnected *DB
func NewPreDB() *PreDB {
	return &PreDB{
		DB: &DB{},
	}
}

// Init init
func (p *PreDB) Init(dbConfig *Config) error {
	db, err := Connect(dbConfig)
	if err != nil {
		return err
	}
	p.inited = true
	p.DB = db
	return nil
}
