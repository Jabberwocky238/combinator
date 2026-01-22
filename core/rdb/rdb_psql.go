package rdb

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"strings"

	"bytes"

	_ "github.com/lib/pq"
)

type PsqlRDB struct {
	db       *sql.DB
	host     string
	port     int
	user     string
	password string
	dbname   string
}

func NewPsqlRDB(host string, port int, user string, password string, dbname string) *PsqlRDB {
	rdb := &PsqlRDB{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbname:   dbname,
	}
	return rdb
}

// Execute executes a DML/DDL statement with optional parameters
func (r *PsqlRDB) Execute(stmt string, args ...any) (lastInsertId int, rowsAffected int, err error) {
	// Convert ? placeholders to $1, $2, etc. for PostgreSQL
	stmt, err = convertPlaceholders(stmt)
	if err != nil {
		return 0, 0, err
	}

	// Validate parameters
	if err := validateParamsPsql(stmt, args); err != nil {
		return 0, 0, err
	}

	result, err := r.db.Exec(stmt, args...)
	if err != nil {
		return 0, 0, err
	}

	rowsAffected64, err := result.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	lastInsertId64, err := result.LastInsertId()
	if err != nil {
		return 0, 0, err
	}

	return int(lastInsertId64), int(rowsAffected64), nil
}

// Query executes a SELECT statement with optional parameters and returns CSV
func (r *PsqlRDB) Query(stmt string, args ...any) ([]byte, error) {
	// Convert ? placeholders to $1, $2, etc. for PostgreSQL
	stmt, err := convertPlaceholders(stmt)
	if err != nil {
		return nil, err
	}

	// Validate parameters
	if err := validateParamsPsql(stmt, args); err != nil {
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
func (r *PsqlRDB) Batch(stmts string) error {
	_, err := exec(r.db, stmts, r.Type())
	return err
}

func (r *PsqlRDB) Start() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		r.host, r.port, r.user, r.password, r.dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	r.db = db
	return nil
}

func (r *PsqlRDB) Type() string {
	return "postgres"
}

// convertPlaceholders converts ? placeholders to $1, $2, etc. for PostgreSQL
func convertPlaceholders(stmt string) (string, error) {
	var result strings.Builder
	paramIndex := 1

	for i := 0; i < len(stmt); i++ {
		if stmt[i] == '?' {
			result.WriteString(fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		} else {
			result.WriteByte(stmt[i])
		}
	}

	return result.String(), nil
}

// validateParamsPsql validates that the number of placeholders matches the number of arguments
func validateParamsPsql(stmt string, args []any) error {
	// Count $1, $2, etc. placeholders
	placeholderCount := 0
	for i := 1; ; i++ {
		placeholder := fmt.Sprintf("$%d", i)
		if strings.Contains(stmt, placeholder) {
			placeholderCount++
		} else {
			break
		}
	}

	if placeholderCount != len(args) {
		return fmt.Errorf("parameter count mismatch: statement has %d placeholders but %d arguments provided", placeholderCount, len(args))
	}
	return nil
}
