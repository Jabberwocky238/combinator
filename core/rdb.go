package combinator

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xwb1989/sqlparser"
)

type SqliteRDB struct {
	db *sql.DB
}

func NewSqliteRDB(url string) (*SqliteRDB, error) {
	sqlite_db, err := sql.Open("sqlite3", url)
	if err != nil {
		return nil, err
	}
	return &SqliteRDB{db: sqlite_db}, nil
}

func (r *SqliteRDB) Execute(stmts string) ([]byte, error) {
	return exec(r.db, stmts)
}

func (r *SqliteRDB) Start() error {
	return nil
}

type PsqlRDB struct {
	db       *sql.DB
	host     string
	port     int
	user     string
	password string
	dbname   string
}

func NewPsqlRDB(host string, port int, user string, password string, dbname string) (*PsqlRDB, error) {
	rdb := &PsqlRDB{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbname:   dbname,
	}
	return rdb, nil
}

func (r *PsqlRDB) Execute(stmts string) ([]byte, error) {
	return exec(r.db, stmts)
}

func (r *PsqlRDB) Start() error {
	connStr := fmt.Sprintf("%s:%d@%s/%s", r.host, r.port, r.user, r.password, r.dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	r.db = db
	return nil
}

func exec(rdb *sql.DB, stmt string) ([]byte, error) {
	// 第一步：拆分语句
	statements := splitStatements(stmt)

	// 第二步：解析语句（带日志）
	nodes := parseStatements(statements)

	// 第三步：在事务中执行所有语句，使用 buffer writer 收集输出
	return executeInTransaction(rdb, nodes, statements)
}

// 分割 SQL 语句（简单按分号分割）
func splitStatements(stmt string) []string {
	var statements []string

	// 简单按分号分割
	parts := strings.Split(stmt, ";")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}

	// 添加调试日志
	fmt.Printf("[DEBUG] Split into %d statements\n", len(statements))
	for i, s := range statements {
		fmt.Printf("[DEBUG] Statement %d: %s\n", i+1, truncateSQL(s, 50))
	}

	return statements
}

// 第二步：解析语句（带日志）
func parseStatements(statements []string) []sqlparser.Statement {
	nodes := make([]sqlparser.Statement, 0, len(statements))

	for i, stmt := range statements {
		node, err := sqlparser.Parse(stmt)
		if err != nil {
			// 解析失败，记录日志但继续处理
			fmt.Printf("[WARN] Statement %d parse failed: %v\n", i+1, err)
			nodes = append(nodes, nil)
			continue
		}

		// 记录日志
		sqlType := getNodeType(node)
		fmt.Printf("[INFO] Statement %d: %s - %s\n", i+1, sqlType, truncateSQL(stmt, 50))

		nodes = append(nodes, node)
	}

	return nodes
}

// 获取节点类型
func getNodeType(node sqlparser.Statement) string {
	if node == nil {
		return "UNKNOWN"
	}

	switch node.(type) {
	case *sqlparser.Select:
		return "DQL"
	case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
		return "DML"
	case *sqlparser.DDL:
		return "DDL"
	default:
		return "OTHER"
	}
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
func executeInTransaction(db *sql.DB, nodes []sqlparser.Statement, statements []string) ([]byte, error) {
	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 使用 buffer 收集所有输出
	var output bytes.Buffer

	// 执行每条语句
	for i, stmt := range statements {
		node := nodes[i]
		sqlType := getNodeType(node)

		fmt.Printf("[INFO] Executing statement %d: %s\n", i+1, sqlType)

		var err error
		switch sqlType {
		case "DQL":
			// 查询：输出 CSV（列头 + 数据）
			err = executeQueryToWriter(tx, stmt, &output)
		case "DML":
			// 修改：输出 JSON（rows_affected, last_insert_id）
			err = executeExecToWriter(tx, stmt, &output)
		case "DDL":
			// 定义：输出 "OK"
			err = executeDDLToWriter(tx, stmt, &output)
		default:
			err = fmt.Errorf("unknown SQL type: %s", sqlType)
		}

		// 如果出错，回滚事务
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("statement %d failed: %w", i+1, err)
		}

		// 如果还有下一条语句，添加换行符分隔
		if i < len(statements)-1 {
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
func executeDDLToWriter(tx *sql.Tx, stmt string, output *bytes.Buffer) error {
	_, err := tx.Exec(stmt)
	if err != nil {
		return err
	}

	output.WriteString("OK")
	return nil
}
