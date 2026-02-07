package rdb

import (
	"database/sql"
	"time"

	common "jabberwocky238/combinator/core/common"

	_ "github.com/lib/pq"
)

var ebpg = EB.With("postgres")

type PsqlRDB struct {
	db   *sql.DB
	core *RDBCore
	dsn  string
}

func NewPsqlRDB(dsn string) *PsqlRDB {
	rdb := &PsqlRDB{
		dsn: dsn,
	}
	return rdb
}

// Execute executes a DML/DDL statement with optional parameters
func (r *PsqlRDB) Exec(stmt string, args ...any) error {
	return r.core.Exec(stmt, args...)
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *PsqlRDB) Query(stmt string, args ...any) ([]byte, error) {
	return r.core.Query(stmt, args...)
}

// Batch executes multiple SQL statements (text format)
func (r *PsqlRDB) Batch(stmts []string, args [][]any) error {
	return r.core.Batch(stmts, args)
}

func (r *PsqlRDB) Start() error {
	if err := r.connect(); err != nil {
		return err
	}
	return nil
}

// connect establishes a new database connection with connection pool settings
func (r *PsqlRDB) connect() error {
	db, err := sql.Open("postgres", r.dsn)
	if err != nil {
		return ebpg.Error("Failed to open postgres connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return ebpg.Error("Failed to ping postgres: %v", err)
	}

	r.db = db
	r.core = &RDBCore{
		db:        db,
		rdbType:   r.Type(),
		reconnect: r.reconnect,
	}

	common.Logger.Infof("PostgreSQL connection established successfully")
	return nil
}

// reconnect closes the old connection and establishes a new one
func (r *PsqlRDB) reconnect() error {
	common.Logger.Warnf("Attempting to reconnect to PostgreSQL...")

	// Close old connection if exists
	if r.db != nil {
		r.db.Close()
	}

	// Establish new connection
	return r.connect()
}

func (r *PsqlRDB) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *PsqlRDB) Type() string {
	return "postgres"
}
