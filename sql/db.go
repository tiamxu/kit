package sql

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tiamxu/kit/log"
)

type DB struct {
	*sqlx.DB
	dbConfig *Config
}

func Connect(dbConfig *Config) (*DB, error) {
	if err := dbConfig.Validate(); err != nil {
		return nil, err
	}

	cfg := *dbConfig
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 10
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 300
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 60
	}

	dsn, err := cfg.Source()
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Connect(cfg.Driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{
		DB:       db,
		dbConfig: &cfg,
	}, nil
}

func (d *DB) TransactCallback(fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) error {
	if fn == nil {
		return nil
	}

	var _tx *sqlx.Tx
	var err error
	var needCommit bool
	if len(tx) > 0 && tx[0] != nil {
		_tx = tx[0]
	} else {
		_tx, err = d.Beginx()
		if err != nil {
			return err
		}
		needCommit = true
		defer func() {
			if needCommit {
				if err != nil {
					if rErr := _tx.Rollback(); rErr != nil {
						log.Errorf("error rolling back transaction: %v", rErr)
					}
				} else {
					if rErr := _tx.Commit(); rErr != nil {
						log.Errorf("error committing transaction: %v", rErr)
						err = rErr
					}
				}
			}
		}()
	}

	err = fn(_tx)
	return err
}

func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

type PreDB struct {
	db     *DB
	mu     sync.Mutex
	inited bool
}

func NewPreDB() *PreDB {
	return &PreDB{}
}

func (p *PreDB) Init(dbConfig *Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inited {
		return nil
	}
	db, err := Connect(dbConfig)
	if err != nil {
		return err
	}
	p.db = db
	p.inited = true
	return nil
}

func (p *PreDB) DB() *DB {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.db
}
