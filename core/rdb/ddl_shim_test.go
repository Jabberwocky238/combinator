package rdb

import (
	"testing"

	"github.com/jabberwocky238/go-sqlparser"
)

// TestDDLShimUniqueness 测试以 SQLite 为标准的 DDL 在两种数据库上的 shim 转换唯一性
func TestDDLShimUniqueness(t *testing.T) {
	tests := []struct {
		name           string
		sqliteSQL      string
		expectPostgres string // 期望的 PostgreSQL 转换结果
		expectSqlite   string // 期望的 SQLite 转换结果（应该保持不变）
	}{
		{
			name: "CREATE TABLE with INTEGER PRIMARY KEY",
			sqliteSQL: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				age INTEGER
			)`,
			expectPostgres: "create table users(id serial primary key autoincrement,name text not null,age integer)",
			expectSqlite:   "create table users(id integer primary key autoincrement,name text not null,age integer)",
		},
		{
			name: "CREATE TABLE without autoincrement",
			sqliteSQL: `CREATE TABLE products (
				id INTEGER,
				name TEXT NOT NULL
			)`,
			expectPostgres: "create table products(id integer,name text not null)",
			expectSqlite:   "create table products(id integer,name text not null)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 解析 SQLite SQL
			ast, err := sqlparser.Parse(tt.sqliteSQL)
			if err != nil {
				t.Fatalf("Failed to parse SQLite SQL: %v", err)
			}

			if len(ast.Statements) == 0 {
				t.Fatal("No statements parsed")
			}

			sqliteNode := ast.Statements[0]
			t.Logf("Original SQLite SQL: %s", sqliteNode.String())
			t.Logf("Statement type: %T", sqliteNode)

			// 测试 PostgreSQL shim
			pgNode := ddlShimPostgres(sqliteNode)
			pgSQL := pgNode.String()
			t.Logf("PostgreSQL transformed SQL: %s", pgSQL)

			if pgSQL != tt.expectPostgres {
				t.Errorf("PostgreSQL shim mismatch:\nExpected: %s\nGot:      %s", tt.expectPostgres, pgSQL)
			}

			// 重新解析以获取新的 node（避免修改原始 node）
			ast2, _ := sqlparser.Parse(tt.sqliteSQL)
			sqliteNode2 := ast2.Statements[0]

			// 测试 SQLite shim（应该保持不变）
			sqliteShimNode := ddlShimSqlite(sqliteNode2)
			sqliteSQL := sqliteShimNode.String()
			t.Logf("SQLite transformed SQL: %s", sqliteSQL)

			if sqliteSQL != tt.expectSqlite {
				t.Errorf("SQLite shim mismatch:\nExpected: %s\nGot:      %s", tt.expectSqlite, sqliteSQL)
			}
		})
	}
}

// TestCreateIndexNotSupported 测试 CREATE INDEX 语句不被 sqlparser 支持
// CREATE INDEX 语句会在 parseStatements 中解析失败，返回 nil node
// 在 executeInTransaction 中会被归类为 default case，返回 "unknown SQL type" 错误
func TestCreateIndexNotSupported(t *testing.T) {
	indexSQL := "CREATE INDEX idx_users_name ON users(name)"

	ast, err := sqlparser.Parse(indexSQL)
	if err != nil {
		t.Logf("Expected: CREATE INDEX is not supported by tablelandnetwork/sqlparser")
		t.Logf("Parse error: %v", err)
		return
	}

	// 如果解析成功（不太可能），记录类型
	if len(ast.Statements) > 0 {
		t.Logf("Unexpected: CREATE INDEX was parsed successfully")
		t.Logf("Statement type: %T", ast.Statements[0])
		t.Logf("Statement SQL: %s", ast.Statements[0].String())
	}
}
