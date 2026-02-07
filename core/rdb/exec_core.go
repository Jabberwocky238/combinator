package rdb

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"

	sqlparser "github.com/jabberwocky238/sqlparser"
)

var ebcore = EB.With("core")

type RDBCore struct {
	db        *sql.DB
	rdbType   string
	reconnect func() error // reconnect callback function
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

// 第一步：解析语句，判断类型（DQL/DML/DDL），并根据数据库类型应用 shim 转换
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
	} else if sqlType == SQL_TYPE_DML || sqlType == SQL_TYPE_DQL {
		var newStmt string
		switch rdbType {
		case "postgres":
			newStmt = shimPlaceholdersPostgres(stmt)
		case "sqlite":
			newStmt = shimPlaceholdersSqlite(stmt)
		default:
			newStmt = stmt
		}
		// 重新解析转换后的语句
		newAst, err := sqlparser.Parse(newStmt)
		if err != nil {
			return nil, SQL_TYPE_UNKNOWN, ebcore.Error("Statement re-parse failed after shimming: %v", err)
		}
		transformedNode = newAst.Statements[0]
	} else {
		fmt.Printf("[WARN] Statement type is unknown, no shim applied\n")
		return nil, sqlType, ebcore.Error("unknown statement type: %T", node)
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

// 第三步：在事务中执行所有语句
func executeInTransaction(db *sql.DB, nodes []sqlparser.Statement, args [][]any, rdbType string) error {
	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行每条语句
	for i, node := range nodes {
		var err error
		stmt := node.String()
		fmt.Printf("[DEBUG] Statement %d: %s\n", i+1, stmt)
		switch node.(type) {
		case *sqlparser.Select:
			// DQL: 查询，输出 CSV（列头 + 数据）
			_, err = tx.Query(stmt, args[i]...)
		case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
			// DML: 修改，输出 JSON（rows_affected, last_insert_id）
			_, err = tx.Exec(stmt, args[i]...)
		case *sqlparser.CreateTable,
			*sqlparser.AlterTable,
			*sqlparser.DropTable,
			*sqlparser.CreateIndex,
			*sqlparser.DropIndex:
			// DDL: 定义，传入 node 和 rdbType
			_, err = tx.Exec(stmt, args[i]...)
		default:
			err = fmt.Errorf("unknown SQL type: %T", node)
		}

		// 如果出错，回滚事务
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("statement %d failed: %w, stmt: %s, args: %v", i+1, err, stmt, args[i])
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *RDBCore) Query(stmt string, args ...any) ([]byte, error) {
	node, rdbType, err := parseStatement(stmt, r.rdbType)
	if err != nil {
		return nil, err
	}
	if rdbType != SQL_TYPE_DQL {
		return nil, fmt.Errorf("not a DQL statement")
	}

	stmt = node.String() // 使用 parse 后的语句，确保占位符已转换
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
	node, _, err := parseStatement(stmt, r.rdbType)
	if err != nil {
		return err
	}

	stmt = node.String() // 使用 parse 后的语句，确保占位符已转换
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
