package rdb

import (
	"fmt"
	"regexp"
	"strings"

	sqlparser "github.com/jabberwocky238/sqlparser"
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

	return node
}

// isAutoIncrementColumn 检查列是否是自增列
func isAutoIncrementColumn(col *sqlparser.ColumnDef) bool {
	// 检查类型是否是 integer/int
	colType := strings.ToLower(col.Type)
	if colType != "integer" && colType != "int" {
		return false
	}

	// 遍历列的约束，检查是否有 PRIMARY KEY
	// 在 SQLite 中，INTEGER PRIMARY KEY 会自动使用 rowid，即使没有 AUTOINCREMENT 关键字
	// 所以我们需要将 INTEGER PRIMARY KEY 转换为 PostgreSQL 的 SERIAL PRIMARY KEY
	for _, constraint := range col.Constraints {
		if _, ok := constraint.(*sqlparser.ColumnConstraintPrimaryKey); ok {
			// 只要是 INTEGER PRIMARY KEY，就转换为 SERIAL
			return true
		}
	}

	return false
}

// ============================================================================
// 参数占位符转换 (Placeholder Shim)
// ============================================================================

// shimPlaceholdersPostgres 将 ? 占位符转换为 PostgreSQL 的 $1, $2, ... 格式
func shimPlaceholdersPostgres(stmt string) string {
	var result strings.Builder
	paramIndex := 1
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]

		// 处理字符串字面量
		if ch == '\'' || ch == '"' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义的引号
				if i+1 < len(stmt) && stmt[i+1] == stringChar {
					result.WriteByte(ch)
					i++
					result.WriteByte(stmt[i])
					continue
				}
				inString = false
			}
			result.WriteByte(ch)
			continue
		}

		// 在字符串内部，直接写入
		if inString {
			result.WriteByte(ch)
			continue
		}

		// 转换 ? 占位符
		if ch == '?' {
			fmt.Fprintf(&result, "$%d", paramIndex)
			paramIndex++
		} else {
			result.WriteByte(ch)
		}
	}

	return result.String()
}

// shimPlaceholdersSqlite SQLite 使用 ? 占位符，保持不变
func shimPlaceholdersSqlite(stmt string) string {
	return stmt
}

// validatePlaceholdersPostgres 验证 PostgreSQL 占位符数量是否匹配参数数量
func validatePlaceholdersPostgres(stmt string, args []any) error {
	// 使用正则表达式匹配 $1, $2, ... 占位符
	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(stmt, -1)

	if len(matches) == 0 && len(args) == 0 {
		return nil
	}

	if len(matches) != len(args) {
		return fmt.Errorf("parameter count mismatch: statement has %d placeholders but %d arguments provided", len(matches), len(args))
	}

	return nil
}

// validatePlaceholdersSqlite 验证 SQLite 占位符数量是否匹配参数数量
func validatePlaceholdersSqlite(stmt string, args []any) error {
	// 计算 ? 占位符数量（排除字符串内的 ?）
	placeholderCount := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]

		// 处理字符串字面量
		if ch == '\'' || ch == '"' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义的引号
				if i+1 < len(stmt) && stmt[i+1] == stringChar {
					i++
					continue
				}
				inString = false
			}
			continue
		}

		// 在字符串外部，计数 ? 占位符
		if !inString && ch == '?' {
			placeholderCount++
		}
	}

	if placeholderCount != len(args) {
		return fmt.Errorf("parameter count mismatch: statement has %d placeholders but %d arguments provided", placeholderCount, len(args))
	}

	return nil
}
