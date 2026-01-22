package rdb

import (
	"strings"

	sqlparser "github.com/jabberwocky238/go-sqlparser"
)

// ddlShimPostgres 将 SQLite 自增语法转换为 PostgreSQL 语法
func ddlShimPostgres(node sqlparser.Statement) sqlparser.Statement {
	if node == nil {
		return node
	}

	createTable, ok := node.(*sqlparser.CreateTable)
	if !ok {
		return node
	}

	// 遍历表定义中的列
	for _, col := range createTable.ColumnsDef {
		// 检查是否是 INTEGER PRIMARY KEY AUTOINCREMENT
		if isAutoIncrementColumn(col) {
			// 转换为 SERIAL
			col.Type = "serial"
			col.Constraints = filterOutAutoIncrement(col.Constraints)
		}
	}

	return createTable
}

func filterOutAutoIncrement(columnConstraint []sqlparser.ColumnConstraint) []sqlparser.ColumnConstraint {
	filtered := make([]sqlparser.ColumnConstraint, 0, len(columnConstraint))
	for _, constraint := range columnConstraint {
		if pk, ok := constraint.(*sqlparser.ColumnConstraintPrimaryKey); ok {
			// 创建一个新的 ColumnConstraintPrimaryKey，去掉 AutoIncrement 标记
			newPK := *pk
			newPK.AutoIncrement = false
			filtered = append(filtered, &newPK)
		} else {
			filtered = append(filtered, constraint)
		}
	}
	return filtered
}

// ddlShimSqlite 保持 SQLite 语法不变
func ddlShimSqlite(node sqlparser.Statement) sqlparser.Statement {
	// SQLite 不需要转换
	return node
}

// isAutoIncrementColumn 检查列是否是自增列
func isAutoIncrementColumn(col *sqlparser.ColumnDef) bool {
	// 检查类型是否是 integer/int
	colType := strings.ToLower(col.Type)
	if colType != "integer" && colType != "int" {
		return false
	}

	// 遍历列的约束，检查是否有 PRIMARY KEY 且带 AUTOINCREMENT
	for _, constraint := range col.Constraints {
		if pk, ok := constraint.(*sqlparser.ColumnConstraintPrimaryKey); ok {
			// 只有当明确标记了 AutoIncrement 时才转换
			if pk.AutoIncrement {
				return true
			}
		}
	}

	return false
}
