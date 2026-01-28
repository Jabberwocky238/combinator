package rdb

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strings"

	sqlparser "github.com/jabberwocky238/sqlparser"
)

var ebcore = EB.With("core")

type RDBCore struct {
	db      *sql.DB
	rdbType string
}

func NewRDBCore(db *sql.DB, rdbType string) *RDBCore {
	return &RDBCore{
		db:      db,
		rdbType: rdbType,
	}
}

type SQLType string

var (
	SQL_TYPE_DQL     SQLType = "DQL"
	SQL_TYPE_DML     SQLType = "DML"
	SQL_TYPE_DDL     SQLType = "DDL"
	SQL_TYPE_UNKNOWN SQLType = "OTHER"
)

func parseStatement(stmt string, rdbType string) (sqlparser.Statement, SQLType, error) {
	ast, err := sqlparser.Parse(stmt)
	if err != nil {
		return nil, SQL_TYPE_UNKNOWN, ebcore.Error("Statement parse failed: %v", err)
	}

	// 新的 sqlparser 返回 AST，包含多个 statements
	if len(ast.Statements) == 0 {
		fmt.Printf("[WARN] Statement has no statements in AST\n")
	} else if len(ast.Statements) > 1 {
		return nil, SQL_TYPE_UNKNOWN, ebcore.Error("multiple statements not supported")
	}

	// 取第一个 statement
	node := ast.Statements[0]

	// 记录日志
	var sqlType SQLType
	switch node.(type) {
	case *sqlparser.Select:
		sqlType = SQL_TYPE_DQL
	case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
		sqlType = SQL_TYPE_DML
	case *sqlparser.CreateTable,
		*sqlparser.AlterTable,
		*sqlparser.DropTable,
		*sqlparser.CreateIndex,
		*sqlparser.DropIndex:
		sqlType = SQL_TYPE_DDL
	default:
		sqlType = SQL_TYPE_UNKNOWN
	}

	// 根据数据库类型应用 shim
	var transformedNode sqlparser.Statement = node
	if sqlType == SQL_TYPE_DDL {
		switch rdbType {
		case "postgres":
			transformedNode = ddlShimPostgres(node)
		case "sqlite":
			transformedNode = ddlShimSqlite(node)
		default:
			transformedNode = node
		}
	}

	fmt.Printf("[INFO] Statement: %s - %s\n", sqlType, transformedNode.String())
	return transformedNode, sqlType, nil
}

// 第二步：解析语句（带日志）
func parseStatements(statements []string, rdbType string) []sqlparser.Statement {
	nodes := make([]sqlparser.Statement, 0, len(statements))

	for i, stmt := range statements {
		node, _, err := parseStatement(stmt, rdbType)
		if err != nil {
			fmt.Printf("[ERROR] Failed to parse statement %d: %v\n", i+1, err)
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// 截断 SQL 用于日志显示
func truncateSQL(sql string, maxLen int) string {
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.TrimSpace(sql)
	if len(sql) > maxLen {
		return sql[:maxLen] + "..."
	}
	return sql
}

// 第三步：在事务中执行所有语句
func executeInTransaction(db *sql.DB, nodes []sqlparser.Statement, args [][]any, rdbType string) error {
	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行每条语句
	for i, node := range nodes {
		fmt.Printf("[INFO] Executing statement %d\n", i+1)

		var err error
		switch node.(type) {
		case *sqlparser.Select:
			// DQL: 查询，输出 CSV（列头 + 数据）
			stmt := node.String()
			err = executeQueryToWriter(tx, stmt, args[i])
		case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
			// DML: 修改，输出 JSON（rows_affected, last_insert_id）
			stmt := node.String()
			err = executeDMLToWriter(tx, stmt, args[i])
		case *sqlparser.CreateTable,
			*sqlparser.AlterTable,
			*sqlparser.DropTable,
			*sqlparser.CreateIndex,
			*sqlparser.DropIndex:
			// DDL: 定义，传入 node 和 rdbType
			err = executeDDLToWriter(tx, node, rdbType)
		default:
			err = fmt.Errorf("unknown SQL type: %T", node)
		}

		// 如果出错，回滚事务
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("statement %d failed: %w", i+1, err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// 在事务中执行查询
func executeQueryToWriter(tx *sql.Tx, stmt string, args []any) error {
	rows, err := tx.Query(stmt, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	_, err = rows.Columns()
	if err != nil {
		return err
	}
	return nil
}

// 在事务中执行
func executeDMLToWriter(tx *sql.Tx, stmt string, args []any) error {
	_, err := tx.Exec(stmt, args...)
	if err != nil {
		return err
	}
	return nil
}

// 在事务中执行 DDL，输出 "OK" 到 writer
func executeDDLToWriter(tx *sql.Tx, node sqlparser.Statement, rdbType string) error {
	// 根据数据库类型应用 shim
	var transformedNode sqlparser.Statement
	switch rdbType {
	case "postgres":
		transformedNode = ddlShimPostgres(node)
	case "sqlite":
		transformedNode = ddlShimSqlite(node)
	default:
		transformedNode = node
	}

	// 从转换后的 node 生成 SQL
	stmt := transformedNode.String()

	// 执行 DDL
	_, err := tx.Exec(stmt)
	if err != nil {
		return err
	}
	return nil
}

func (r *RDBCore) Query(stmt string, args ...any) ([]byte, error) {
	_, rdbType, err := parseStatement(stmt, r.rdbType)
	if err != nil {
		return nil, err
	}
	if rdbType != SQL_TYPE_DQL {
		return nil, fmt.Errorf("not a DQL statement")
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

// Execute executes a DML/DDL statement with optional parameters
func (r *RDBCore) Exec(stmt string, args ...any) error {
	_, _, err := parseStatement(stmt, r.rdbType)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(stmt, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *RDBCore) Batch(stmts []string, args [][]any) error {
	// 第二步：解析语句（带日志）
	nodes := parseStatements(stmts, r.rdbType)

	// 第三步：在事务中执行所有语句，使用 buffer writer 收集输出
	return executeInTransaction(r.db, nodes, args, r.rdbType)
}
