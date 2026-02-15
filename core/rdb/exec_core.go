package rdb

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"

	common "jabberwocky238/combinator/core/common"

	sqlparser "github.com/jabberwocky238/sqlparser"
)

type RDBCore struct {
	db        *sql.DB
	rdbType   string
	reconnect func() error // reconnect callback function
	log       *common.NamespacedLogger
}

type SQLType string

var (
	SQL_TYPE_DQL     SQLType = "DQL"
	SQL_TYPE_DML     SQLType = "DML"
	SQL_TYPE_DDL     SQLType = "DDL"
	SQL_TYPE_UNKNOWN SQLType = "OTHER"
)

// 第一步：解析语句，判断类型（DQL/DML/DDL），并根据数据库类型应用 shim 转换
func (r *RDBCore) parseStatement(stmt string) (sqlparser.Statement, SQLType, error) {
	ast, err := sqlparser.Parse(stmt)
	if err != nil {
		return nil, SQL_TYPE_UNKNOWN, r.log.NewError("Statement parse failed: %v", err)
	}

	// 新的 sqlparser 返回 AST，包含多个 statements
	if len(ast.Statements) == 0 {
		r.log.Warnf("Statement has no statements in AST")
	} else if len(ast.Statements) > 1 {
		return nil, SQL_TYPE_UNKNOWN, r.log.NewError("multiple statements not supported")
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
		switch r.rdbType {
		case "postgres":
			transformedNode = ddlShimPostgres(node)
		case "sqlite":
			transformedNode = ddlShimSqlite(node)
		default:
			transformedNode = node
		}
	} else if sqlType == SQL_TYPE_DML || sqlType == SQL_TYPE_DQL {
		var newStmt string
		switch r.rdbType {
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
			return nil, SQL_TYPE_UNKNOWN, r.log.NewError("Statement re-parse failed after shimming: %v", err)
		}
		transformedNode = newAst.Statements[0]
	} else {
		r.log.Warnf("Statement type is unknown, no shim applied")
		return nil, sqlType, r.log.NewError("unknown statement type: %T", node)
	}

	r.log.Infof("Statement: %s - %s", sqlType, transformedNode.String())
	return transformedNode, sqlType, nil
}

// 第二步：解析语句（带日志）
func (r *RDBCore) parseStatements(statements []string) []sqlparser.Statement {
	nodes := make([]sqlparser.Statement, 0, len(statements))
	for i, stmt := range statements {
		node, _, err := r.parseStatement(stmt)
		if err != nil {
			r.log.Errorf("Failed to parse statement %d: %v", i+1, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// 第三步：在事务中执行所有语句
func (r *RDBCore) executeInTransaction(nodes []sqlparser.Statement, args [][]any) error {
	// 开启事务
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行每条语句
	for i, node := range nodes {
		var err error
		stmt := node.String()
		r.log.Debugf("Statement %d: %s", i+1, stmt)
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

func (r *RDBCore) Query(stmt string, args ...any) (io.ReadCloser, error) {
	node, rdbType, err := r.parseStatement(stmt)
	if err != nil {
		return nil, err
	}
	if rdbType != SQL_TYPE_DQL {
		return nil, fmt.Errorf("not a DQL statement")
	}

	stmt = node.String()
	rows, err := r.db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer rows.Close()
		defer pw.Close()

		bufWriter := bufio.NewWriter(pw)
		writer := csv.NewWriter(bufWriter)

		writer.Write(columns)
		writer.Flush()
		bufWriter.Flush()

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				pw.CloseWithError(err)
				return
			}

			record := make([]string, len(columns))
			for i, val := range values {
				if val == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", val)
				}
			}

			writer.Write(record)
			writer.Flush()
			bufWriter.Flush()
		}

		if err := rows.Err(); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

// Execute executes a DML/DDL statement with optional parameters
func (r *RDBCore) Exec(stmt string, args ...any) error {
	node, _, err := r.parseStatement(stmt)
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
	nodes := r.parseStatements(stmts)
	// 第三步：在事务中执行所有语句，使用 buffer writer 收集输出
	return r.executeInTransaction(nodes, args)
}
