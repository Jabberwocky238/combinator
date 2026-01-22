package rdb

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"

	"bytes"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type SqliteRDB struct {
	db  *sql.DB
	url string
}

func NewSqliteRDB(url string) *SqliteRDB {
	return &SqliteRDB{url: url}
}

func (r *SqliteRDB) Execute(stmt string, args ...any) ([]byte, error) {
	result, err := r.db.Exec(stmt, args)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	resultMap := map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertId,
	}

	jsonData, err := json.Marshal(resultMap)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (r *SqliteRDB) Query(stmt string, args ...any) ([]byte, error) {
	rows, err := r.db.Query(stmt, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// 使用 CSV writer
	writer := csv.NewWriter(&buf)

	// 写入列头
	if err := writer.Write(columns); err != nil {
		return nil, err
	}

	// 写入数据行
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 转换为字符串数组
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

func (r *SqliteRDB) Batch(stmts string) error {
	_, err := exec(r.db, stmts, r.Type())
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

func (r *PsqlRDB) Execute(stmts string) ([]byte, error) {
	return exec(r.db, stmts, r.Type())
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
