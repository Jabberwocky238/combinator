package rdb

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	sqlparser "github.com/jabberwocky238/go-sqlparser"
)

func exec(rdb *sql.DB, stmts []string, rdbType string) ([]byte, error) {
	// 第二步：解析语句（带日志）
	nodes := parseStatements(stmts)

	// 第三步：在事务中执行所有语句，使用 buffer writer 收集输出
	return executeInTransaction(rdb, nodes, rdbType)
}

// 第二步：解析语句（带日志）
func parseStatements(statements []string) []sqlparser.Statement {
	nodes := make([]sqlparser.Statement, 0, len(statements))

	for i, stmt := range statements {
		ast, err := sqlparser.Parse(stmt)
		if err != nil {
			// 解析失败，记录日志但继续处理
			fmt.Printf("[WARN] Statement %d parse failed: %v\n", i+1, err)
			nodes = append(nodes, nil)
			continue
		}

		// 新的 sqlparser 返回 AST，包含多个 statements
		if len(ast.Statements) == 0 {
			fmt.Printf("[WARN] Statement %d: no statements in AST\n", i+1)
			nodes = append(nodes, nil)
			continue
		}

		// 取第一个 statement
		node := ast.Statements[0]

		// 记录日志
		var sqlType string
		switch node.(type) {
		case *sqlparser.Select:
			sqlType = "DQL"
		case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
			sqlType = "DML"
		case *sqlparser.CreateTable, *sqlparser.AlterTable:
			sqlType = "DDL"
		default:
			sqlType = "OTHER"
		}
		fmt.Printf("[INFO] Statement %d: %s - %s\n", i+1, sqlType, truncateSQL(stmt, 50))

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
func executeInTransaction(db *sql.DB, nodes []sqlparser.Statement, rdbType string) ([]byte, error) {
	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 使用 buffer 收集所有输出
	var output bytes.Buffer

	// 执行每条语句
	for i, node := range nodes {
		fmt.Printf("[INFO] Executing statement %d\n", i+1)

		var err error
		switch node.(type) {
		case *sqlparser.Select:
			// DQL: 查询，输出 CSV（列头 + 数据）
			stmt := node.String()
			err = executeQueryToWriter(tx, stmt, &output)
		case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
			// DML: 修改，输出 JSON（rows_affected, last_insert_id）
			stmt := node.String()
			err = executeExecToWriter(tx, stmt, &output)
		case *sqlparser.CreateTable, *sqlparser.AlterTable, *sqlparser.DropTable:
			// DDL: 定义，传入 node 和 rdbType
			err = executeDDLToWriter(tx, node, rdbType, &output)
		default:
			err = fmt.Errorf("unknown SQL type")
		}

		// 如果出错，回滚事务
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("statement %d failed: %w", i+1, err)
		}

		// 如果还有下一条语句，添加换行符分隔
		if i < len(nodes)-1 {
			output.WriteString("\n\n")
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return output.Bytes(), nil
}

// 在事务中执行查询，输出 CSV 到 writer
func executeQueryToWriter(tx *sql.Tx, stmt string, output *bytes.Buffer) error {
	rows, err := tx.Query(stmt)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// 使用 CSV writer
	writer := csv.NewWriter(output)

	// 写入列头
	if err := writer.Write(columns); err != nil {
		return err
	}

	// 写入数据行
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
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
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// 在事务中执行 DML，输出 JSON 到 writer
func executeExecToWriter(tx *sql.Tx, stmt string, output *bytes.Buffer) error {
	result, err := tx.Exec(stmt)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	resultMap := map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertId,
	}

	jsonData, err := json.Marshal(resultMap)
	if err != nil {
		return err
	}

	output.Write(jsonData)
	return nil
}

// 在事务中执行 DDL，输出 "OK" 到 writer
func executeDDLToWriter(tx *sql.Tx, node sqlparser.Statement, rdbType string, output *bytes.Buffer) error {
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

	output.WriteString("OK")
	return nil
}
