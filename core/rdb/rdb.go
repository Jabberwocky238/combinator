package rdb

import (
	"database/sql"
	"fmt"

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

func (r *SqliteRDB) Execute(stmts string) ([]byte, error) {
	return exec(r.db, stmts, r.Type())
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
