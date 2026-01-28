package rdb

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	common "jabberwocky238/combinator/core/common"
	"strings"

	"bytes"

	_ "github.com/mattn/go-sqlite3"
)

var eb = EB.With("sqlite")

type SqliteRDB struct {
	db  *sql.DB
	url string
}

func NewSqliteRDB(url string) *SqliteRDB {
	return &SqliteRDB{url: url}
}

// Execute executes a DML/DDL statement with optional parameters
func (r *SqliteRDB) Execute(stmt string, args ...any) (int, error) {
	// Validate parameters
	if err := validateParams(stmt, args); err != nil {
		return 0, err
	}

	result, err := r.db.Exec(stmt, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *SqliteRDB) Query(stmt string, args ...any) ([]byte, error) {
	// Validate parameters
	if err := validateParams(stmt, args); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// Use CSV writer
	writer := csv.NewWriter(&buf)

	// Write column headers
	if err := writer.Write(columns); err != nil {
		return nil, err
	}

	// Write data rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Convert to string array
		record := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// Batch executes multiple SQL statements (text format)
func (r *SqliteRDB) Batch(stmts []string) error {
	_, err := exec(r.db, stmts, r.Type())
	if err != nil {
		common.Logger.Errorf("Batch execution error: %v", err)
		return eb.Error("Batch execution error: %v", err)
	}
	return err
}

func (r *SqliteRDB) Start() error {
	sqlite_db, err := sql.Open("sqlite3", r.url)
	if err != nil {
		return err
	}
	r.db = sqlite_db
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
