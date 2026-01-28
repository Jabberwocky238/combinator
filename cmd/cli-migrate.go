package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	migrateRdbIndex string
	migrationDir    string
	apiAddr         string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "执行数据库迁移",
	Run:   runMigrate,
}

func init() {
	migrateCmd.Flags().StringVarP(&migrateRdbIndex, "index", "i", "", "RDB 实例 ID (必填)")
	migrateCmd.Flags().StringVarP(&migrationDir, "migration-dir", "d", "./migrations", "migrations 文件夹路径")
	migrateCmd.Flags().StringVar(&apiAddr, "api", "localhost:8899", "Combinator API 服务器地址")
	migrateCmd.MarkFlagRequired("index")
}

func runMigrate(cmd *cobra.Command, args []string) {
	if migrateRdbIndex == "" {
		fmt.Println("必须指定 RDB 实例 ID，使用 -i 或 --index 参数")
		return
	}

	fmt.Printf("RDB 实例 ID: %s\n", migrateRdbIndex)
	fmt.Printf("Migrations 目录: %s\n", migrationDir)
	fmt.Printf("API 地址: %s\n\n", apiAddr)

	if err := ensureMigrationTable(); err != nil {
		fmt.Printf("创建迁移表失败: %v\n", err)
		return
	}

	executed, err := getExecutedMigrations()
	if err != nil {
		fmt.Printf("获取已执行迁移失败: %v\n", err)
		return
	}

	sqlFiles, err := collectSQLFiles(migrationDir)
	if err != nil {
		fmt.Printf("读取 migrations 目录失败: %v\n", err)
		return
	}

	if len(sqlFiles) == 0 {
		fmt.Println("没有找到 SQL 文件")
		return
	}

	fmt.Printf("找到 %d 个 SQL 文件\n", len(sqlFiles))

	executedCount := 0
	for _, sqlFile := range sqlFiles {
		fileName := filepath.Base(sqlFile)
		fmt.Printf("执行迁移: %s ... ", fileName)
		if executed[fileName] {
			fmt.Printf("- %s (已执行，跳过)\n", fileName)
			continue
		}

		if err := executeSQLFile(sqlFile); err != nil {
			fmt.Printf("执行 %s 失败: %v\n", fileName, err)
			return
		}

		if err := recordMigration(fileName); err != nil {
			fmt.Printf("记录迁移 %s 失败: %v\n", fileName, err)
			return
		}

		fmt.Printf("✓ %s\n", fileName)
		executedCount++
	}

	if executedCount == 0 {
		fmt.Println("\n没有新的迁移需要执行")
	} else {
		fmt.Printf("\n迁移完成，执行了 %d 个文件\n", executedCount)
	}
}

func collectSQLFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func executeSQLFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return executeSQL(string(content))
}

func ensureMigrationTable() error {
	sql := `CREATE TABLE IF NOT EXISTS combinator_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		migration TEXT NOT NULL UNIQUE
	);`
	return executeSQL(sql)
}

func getExecutedMigrations() (map[string]bool, error) {
	sql := `SELECT migration FROM combinator_migrations;`
	body, err := querySQL(sql)
	if err != nil {
		return nil, err
	}

	executed := make(map[string]bool)
	reader := csv.NewReader(strings.NewReader(body))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for i, record := range records {
		if i == 0 || len(record) == 0 {
			continue
		}
		executed[record[0]] = true
	}
	return executed, nil
}

func recordMigration(name string) error {
	sql := fmt.Sprintf(`INSERT INTO "combinator_migrations" (migration) VALUES ('%s');`, name)
	return executeSQL(sql)
}

func executeSQL(sql string) error {
	url := fmt.Sprintf("http://%s/rdb/batch", apiAddr)
	// split statements by semicolon
	var stmtsList []string
	statements := strings.Split(sql, ";")
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed != "" {
			stmtsList = append(stmtsList, trimmed+";")
		}
	}
	reqBody, err := json.Marshal(stmtsList)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Combinator-RDB-ID", migrateRdbIndex)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func querySQL(sql string) (string, error) {
	url := fmt.Sprintf("http://%s/rdb/query", apiAddr)
	type QueryRequest struct {
		SQL  string `json:"stmt"`
		Args []any  `json:"args"`
	}

	var reqBody bytes.Buffer
	err := json.NewEncoder(&reqBody).Encode(QueryRequest{
		SQL:  sql,
		Args: []any{},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", url, &reqBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Combinator-RDB-ID", migrateRdbIndex)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
