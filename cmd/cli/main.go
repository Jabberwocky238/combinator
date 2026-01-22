package main

/*
CLI Command Structure - All Possible Commands

=== Manager Commands (管理 Physical RDBs) ===

1. manager rdb list
   - 列出所有 Physical RDBs (superuser 连接)
   - 示例: combinator-cli manager rdb list

2. manager rdb add <id> <url>
   - 添加新的 Physical RDB (仅支持 PostgreSQL)
   - PostgreSQL: combinator-cli manager rdb add psql-1 "postgres://postgres:pass@localhost:5432/postgres"
   - 注意: SQLite 不支持 Physical RDB (没有管理员概念)

3. manager rdb remove <id>
   - 移除 Physical RDB
   - 示例: combinator-cli manager rdb remove psql-1

4. manager rdb databases <id>
   - 列出 Physical RDB 中的所有数据库
   - 示例: combinator-cli manager rdb databases psql-1

=== RDB Commands (管理 Logical RDBs) ===

5. rdb list
   - 列出所有 Logical RDBs (Processor 使用的受限连接)
   - 示例: combinator-cli rdb list

6. rdb add <id> <url>
   - 添加新的 Logical RDB 到 Processor
   - PostgreSQL: combinator-cli rdb add app-db-1 "postgres://app_user:pass@localhost:5432/myapp"
   - SQLite: combinator-cli rdb add test-db "sqlite:///path/to/db.db" 或 "sqlite://:memory:"

7. rdb remove <id>
   - 从 Processor 移除 Logical RDB
   - 示例: combinator-cli rdb remove app-db-1

=== Database Creation Commands ===

8. create <physical_rdb_id> <url>
   - 使用 Physical RDB 创建新数据库、用户并授权 (一次性操作)
   - 从 URL 中解析: 数据库名、用户名、密码
   - 示例: combinator-cli create psql-main "postgres://myapp_user:myapp_pass@localhost:5432/myapp"
   - 这会自动:
     a. 从 URL 解析出 dbname=myapp, username=myapp_user, password=myapp_pass
     b. 在 psql-main 上创建数据库 myapp
     c. 创建用户 myapp_user
     d. 授予 myapp_user 对 myapp 的所有权限
     e. 将这个 URL 记录到配置文件的 logical_rdbs 中
   - 注意: 每次使用时都会重新 parse URL

=== Config Commands ===

9. config show
   - 显示当前配置文件内容
   - 示例: combinator-cli config show

10. config reload
    - 重新加载配置文件 (热更新)
    - 示例: combinator-cli config reload

11. config save
    - 保存当前配置到文件
    - 示例: combinator-cli config save

=== KV Commands (未来扩展) ===

12. manager kv list
    - 列出所有 Physical KVs
    - 示例: combinator-cli manager kv list

13. kv list
    - 列出所有 Logical KVs
    - 示例: combinator-cli kv list

=== Health Check ===

14. health
    - 检查 Manager 健康状态
    - 示例: combinator-cli health

15. ping
    - Ping Manager 服务
    - 示例: combinator-cli ping

*/

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

var managerURL = "http://localhost:10086"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "manager":
		handleManager()
	case "rdb":
		handleRDB()
	case "create":
		handleCreate()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: combinator-cli <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  manager rdb list                                    - List physical RDBs")
	fmt.Println("  rdb list                                            - List logical RDBs")
	fmt.Println("  rdb databases <rdb_id>                              - List databases in RDB")
	fmt.Println("  create <rdb_id> <db_name> <username> <password>     - Create logical RDB")
}

// handleManager handles manager commands
func handleManager() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: combinator-cli manager rdb list")
		os.Exit(1)
	}

	subcommand := os.Args[2]

	if subcommand == "rdb" {
		if len(os.Args) < 4 {
			fmt.Println("Usage: combinator-cli manager rdb list")
			os.Exit(1)
		}
		if os.Args[3] == "list" {
			listPhysicalRDBs()
		} else {
			fmt.Printf("Unknown manager rdb subcommand: %s\n", os.Args[3])
			os.Exit(1)
		}
	} else {
		fmt.Printf("Unknown manager subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// handleRDB handles rdb commands
func handleRDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: combinator-cli rdb <list|databases>")
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		listLogicalRDBs()
	case "databases":
		if len(os.Args) < 4 {
			fmt.Println("Usage: combinator-cli rdb databases <rdb_id>")
			os.Exit(1)
		}
		listDatabases(os.Args[3])
	default:
		fmt.Printf("Unknown rdb subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// handleCreate handles create command
func handleCreate() {
	if len(os.Args) < 6 {
		fmt.Println("Usage: combinator-cli create <rdb_id> <db_name> <username> <password>")
		os.Exit(1)
	}

	createLogicalRDB(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
}

// listPhysicalRDBs lists all physical RDBs (Manager)
func listPhysicalRDBs() {
	resp, err := http.Get(managerURL + "/api/physical/rdbs")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

// listLogicalRDBs lists all logical RDBs (Processor)
func listLogicalRDBs() {
	// TODO: This will call Processor API when implemented
	fmt.Println("Logical RDB list not yet implemented")
}

// listDatabases lists databases in a physical RDB
func listDatabases(rdbID string) {
	url := fmt.Sprintf("%s/api/physical/rdbs/%s/databases", managerURL, rdbID)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

// createLogicalRDB creates a logical RDB (db + user + grant)
func createLogicalRDB(rdbID, dbName, username, password string) {
	url := fmt.Sprintf("%s/api/logical/rdb/create?rdb_id=%s&db_name=%s&username=%s&password=%s",
		managerURL, rdbID, dbName, username, password)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
