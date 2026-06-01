package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	kiterrors "github.com/tiamxu/kit/errors"
	"github.com/tiamxu/kit/log"
)

type DB struct {
	*sqlx.DB
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
		return nil, kiterrors.Wrap("SQL_CONNECT", "failed to connect to database", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

	if err := db.Ping(); err != nil {
		return nil, kiterrors.Wrap("SQL_PING", "failed to ping database", err)
	}

	return &DB{
		DB: db,
	}, nil
}

func (d *DB) TransactCallback(fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) (err error) {
	if fn == nil {
		return nil
	}

	var _tx *sqlx.Tx
	var needCommit bool
	if len(tx) > 0 && tx[0] != nil {
		_tx = tx[0]
	} else {
		_tx, err = d.Beginx()
		if err != nil {
			return err
		}
		needCommit = true
	}

	// panic safe
	defer func() {
		if r := recover(); r != nil {
			if needCommit {
				_ = _tx.Rollback()
			}
			err = kiterrors.Wrap("SQL_TRANSACTION", "transaction panic recovered", fmt.Errorf("%v", r))
			return
		}
		if needCommit {
			if err != nil {
				if rErr := _tx.Rollback(); rErr != nil {
					log.Errorf("error rolling back transaction: %v", rErr)
				}
			} else {
				if rErr := _tx.Commit(); rErr != nil {
					log.Errorf("error committing transaction: %v", rErr)
					err = kiterrors.Wrap("SQL_COMMIT", "failed to commit transaction", rErr)
				}
			}
		}
	}()

	err = fn(_tx)
	return err
}

func (d *DB) TransactCallbackCtx(ctx context.Context, fn func(*sqlx.Tx) error, tx ...*sqlx.Tx) (err error) {
	if fn == nil {
		return nil
	}

	var _tx *sqlx.Tx
	var needCommit bool
	if len(tx) > 0 && tx[0] != nil {
		_tx = tx[0]
	} else {
		_tx, err = d.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		needCommit = true
	}

	// panic safe
	defer func() {
		if r := recover(); r != nil {
			if needCommit {
				_ = _tx.Rollback()
			}
			err = kiterrors.Wrap("SQL_TRANSACTION", "transaction panic recovered", fmt.Errorf("%v", r))
			return
		}
		if needCommit {
			if err != nil {
				if rErr := _tx.Rollback(); rErr != nil {
					log.Errorf("error rolling back transaction: %v", rErr)
				}
			} else {
				if rErr := _tx.Commit(); rErr != nil {
					log.Errorf("error committing transaction: %v", rErr)
					err = kiterrors.Wrap("SQL_COMMIT", "failed to commit transaction", rErr)
				}
			}
		}
	}()

	err = fn(_tx)
	return err
}

func IsNoRows(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}

func (d *DB) Stats() sql.DBStats {
	return d.DB.Stats()
}

type PreDB struct {
	db     *DB
	mu     sync.RWMutex
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
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.inited {
		return nil
	}
	return p.db
}