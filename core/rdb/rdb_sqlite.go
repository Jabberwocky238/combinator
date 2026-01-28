package rdb

import (
	"database/sql"
	"fmt"
	common "jabberwocky238/combinator/core/common"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
	// Validate parameters
	var err error
	if err = validateParams(stmt, args); err != nil {
		return err
	}
	fmt.Println("[INFO] Executing statement:", stmt)

	err = r.core.Exec(stmt, args...)
	if err != nil {
		return err
	}

	return nil
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *SqliteRDB) Query(stmt string, args ...any) ([]byte, error) {
	// Validate parameters
	if err := validateParams(stmt, args); err != nil {
		return nil, err
	}

	data, err := r.core.Query(stmt, args...)
	if err != nil {
		return nil, err
	}

	return data, nil
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
	sqlite_db, err := sql.Open("sqlite3", r.url)
	if err != nil {
		return err
	}
	r.db = sqlite_db
	r.core = &RDBCore{
		db:      sqlite_db,
		rdbType: r.Type(),
	}
	return nil
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

// validateParams validates that the number of placeholders matches the number of arguments
func validateParams(stmt string, args []any) error {
	placeholderCount := strings.Count(stmt, "?")
	if placeholderCount != len(args) {
		return fmt.Errorf("parameter count mismatch: statement has %d placeholders but %d arguments provided", placeholderCount, len(args))
	}
	return nil
}
