package sql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/tiamxu/kit/log"

	_ "github.com/ClickHouse/clickhouse-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	*sqlx.DB
	dbConfig *Config
}

// Connect to a database and verify with a ping.
func Connect(dbConfig *Config) (*DB, error) {
	if dbConfig.MaxOpenConns <= 0 {
		dbConfig.MaxOpenConns = 10
	}
	if dbConfig.MaxIdleConns <= 0 {
		dbConfig.MaxIdleConns = 5
	}
	if dbConfig.ConnMaxLifetime <= 0 {
		dbConfig.ConnMaxLifetime = 300 // Set a default value (in seconds)
	}
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
func (d *DB) TransactCallback(fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) error {
	if fn == nil {
		return nil
	}

	var _tx *sqlx.Tx
	if len(tx) > 0 {
		_tx = tx[0]
	}
	if _tx == nil {
		_tx, err := d.Beginx()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				if rErr := _tx.Rollback(); rErr != nil {
					// Log the rollback error
					log.Printf("Error rolling back transaction: %v", rErr)
				}
			} else {
				if rErr := _tx.Commit(); rErr != nil {
					// Log the commit error
					log.Printf("Error committing transaction: %v", rErr)
				}
			}
		}()
	}
	return fn(_tx)
}

var ErrNoRows = sql.ErrNoRows

// IsNoRows checks if the error is a "no rows" error, supporting custom error types.
func IsNoRows(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
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
	if p.inited {
		return nil // Prevent re-initializing if already initialized
	}
	db, err := Connect(dbConfig)
	if err != nil {
		return err
	}
	p.inited = true
	p.DB = db
	return nil
}
