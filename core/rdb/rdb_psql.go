package rdb

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

var ebpg = EB.With("postgres")

type PsqlRDB struct {
	db       *sql.DB
	core     *RDBCore
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
func (r *PsqlRDB) Exec(stmt string, args ...any) error {
	// Convert ? placeholders to $1, $2, etc. for PostgreSQL
	stmt, err := convertPlaceholders(stmt)
	log := ebpg.String("Converted statement: %s\n", stmt)
	fmt.Println(log)
	if err != nil {
		return err
	}

	// Validate parameters
	if err := validateParamsPsql(stmt, args); err != nil {
		return err
	}

	err = r.core.Exec(stmt, args...)
	if err != nil {
		return err
	}
	return nil
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
	data, err := r.core.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Batch executes multiple SQL statements (text format)
func (r *PsqlRDB) Batch(stmts []string, args [][]any) error {
	// Convert ? placeholders to $1, $2, etc. for PostgreSQL
	for i, stmt := range stmts {
		convertedStmt, err := convertPlaceholders(stmt)
		log := ebpg.String("Converted statement: %s\n", convertedStmt)
		fmt.Println(log)
		if err != nil {
			return err
		}
		stmts[i] = convertedStmt
	}

	for i, stmt := range stmts {
		// Validate parameters
		if err := validateParamsPsql(stmt, args[i]); err != nil {
			return err
		}
	}

	err := r.core.Batch(stmts, args)
	if err != nil {
		return err
	}
	return nil
}

func (r *PsqlRDB) Start() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		r.host, r.port, r.user, r.password, r.dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	r.db = db
	r.core = &RDBCore{
		db:      db,
		rdbType: r.Type(),
	}
	return nil
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

// convertPlaceholders converts ? placeholders to $1, $2, etc. for PostgreSQL
func convertPlaceholders(stmt string) (string, error) {
	var result strings.Builder
	paramIndex := 1

	for i := 0; i < len(stmt); i++ {
		if stmt[i] == '?' {
			fmt.Fprintf(&result, "$%d", paramIndex)
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
