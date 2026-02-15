package rdb

import (
	"database/sql"
	"io"

	common "jabberwocky238/combinator/core/common"

	_ "modernc.org/sqlite"
)

func init() {
	RegisterRDBFactory("sqlite", func(parsed *ParsedRDBURL, log *common.NamespacedLogger) (common.RDB, error) {
		return NewSqliteRDB(parsed.Path, log.With("sqlite")), nil
	})
}

type SqliteRDB struct {
	db   *sql.DB
	core *RDBCore
	url  string
	log  *common.NamespacedLogger
}

func NewSqliteRDB(url string, log *common.NamespacedLogger) *SqliteRDB {
	return &SqliteRDB{
		url: url,
		log: log,
	}
}

// Execute executes a DML/DDL statement with optional parameters
func (r *SqliteRDB) Exec(stmt string, args ...any) error {
	return r.core.Exec(stmt, args...)
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *SqliteRDB) Query(stmt string, args ...any) (io.ReadCloser, error) {
	return r.core.Query(stmt, args...)
}

// Batch executes multiple SQL statements (text format)
func (r *SqliteRDB) Batch(stmts []string, args [][]any) error {
	err := r.core.Batch(stmts, args)
	if err != nil {
		r.log.Errorf("Batch execution error: %v", err)
		return r.log.NewError("Batch execution error: %v", err)
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
	r.log.Infof("Connecting to SQLite with path: %s", r.url)

	sqlite_db, err := sql.Open("sqlite", r.url)
	if err != nil {
		return r.log.NewError("Failed to open sqlite connection: %v", err)
	}

	// Test the connection
	if err := sqlite_db.Ping(); err != nil {
		sqlite_db.Close()
		return r.log.NewError("Failed to ping sqlite: %v", err)
	}

	r.db = sqlite_db
	r.core = &RDBCore{
		db:        sqlite_db,
		rdbType:   r.Type(),
		reconnect: r.reconnect,
		log:       r.log,
	}

	r.log.Infof("SQLite connection established successfully")
	return nil
}

// reconnect closes the old connection and establishes a new one
func (r *SqliteRDB) reconnect() error {
	r.log.Warnf("Attempting to reconnect to SQLite...")

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
