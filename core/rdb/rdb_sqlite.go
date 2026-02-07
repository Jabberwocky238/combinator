package rdb

import (
	"database/sql"

	common "jabberwocky238/combinator/core/common"

	_ "modernc.org/sqlite"
)

var ebsqlite = EB.With("sqlite")

type SqliteRDB struct {
	db   *sql.DB
	core *RDBCore
	url  string
}

func NewSqliteRDB(url string) *SqliteRDB {
	return &SqliteRDB{url: url}
}

// Execute executes a DML/DDL statement with optional parameters
func (r *SqliteRDB) Exec(stmt string, args ...any) error {
	return r.core.Exec(stmt, args...)
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *SqliteRDB) Query(stmt string, args ...any) ([]byte, error) {
	return r.core.Query(stmt, args...)
}

// Batch executes multiple SQL statements (text format)
func (r *SqliteRDB) Batch(stmts []string, args [][]any) error {
	err := r.core.Batch(stmts, args)
	if err != nil {
		common.Logger.Errorf("Batch execution error: %v", err)
		return ebsqlite.Error("Batch execution error: %v", err)
	}
	return err
}

func (r *SqliteRDB) Start() error {
	if err := r.connect(); err != nil {
		return err
	}
	return nil
}

// connect establishes a new database connection with connection pool settings
func (r *SqliteRDB) connect() error {
	sqlite_db, err := sql.Open("sqlite", r.url)
	if err != nil {
		return ebsqlite.Error("Failed to open sqlite connection: %v", err)
	}

	// Configure connection pool for SQLite
	sqlite_db.SetMaxOpenConns(1) // SQLite works best with single connection
	sqlite_db.SetMaxIdleConns(1)
	sqlite_db.SetConnMaxLifetime(0) // No limit for SQLite

	// Test the connection
	if err := sqlite_db.Ping(); err != nil {
		sqlite_db.Close()
		return ebsqlite.Error("Failed to ping sqlite: %v", err)
	}

	r.db = sqlite_db
	r.core = &RDBCore{
		db:        sqlite_db,
		rdbType:   r.Type(),
		reconnect: r.reconnect,
	}

	common.Logger.Infof("SQLite connection established successfully")
	return nil
}

// reconnect closes the old connection and establishes a new one
func (r *SqliteRDB) reconnect() error {
	common.Logger.Warnf("Attempting to reconnect to SQLite...")

	// Close old connection if exists
	if r.db != nil {
		r.db.Close()
	}

	// Establish new connection
	return r.connect()
}

func (r *SqliteRDB) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *SqliteRDB) Type() string {
	return "sqlite"
}
