# SQL 执行架构设计

## 概述
用户发送 SQL 请求后，系统按照以下三个步骤处理：

## 处理流程

### 第一步：语句拆分 (Split Statements)
**函数**: `splitStatements(sql string) []string`

- 接收原始 SQL 字符串
- 使用 SQL parser 按分号拆分成多条独立的 SQL 语句
- 返回 SQL 语句列表 `[]string`

**输入**:
```sql
CREATE TABLE users (id INT); INSERT INTO users VALUES (1); SELECT * FROM users;
```

**输出**:
```go
[]string{
    "CREATE TABLE users (id INT)",
    "INSERT INTO users VALUES (1)",
    "SELECT * FROM users",
}
```

---

### 第二步：语句解析 (Parse Statements)
**函数**: `parseStatements(statements []string) []sqlparser.Statement`

- 接收 SQL 语句列表
- 对每条语句使用 sqlparser 进行解析
- 得到 SQL AST 节点列表
- **日志输出**：记录每条语句的类型和解析结果

**输入**: `[]string` (SQL 语句列表)

**输出**: `[]sqlparser.Statement` (SQL AST 节点列表)

**日志示例**:
```
[INFO] Statement 1: DDL - CREATE TABLE
[INFO] Statement 2: DML - INSERT
[INFO] Statement 3: DQL - SELECT
```

---

### 第三步：执行并收集结果 (Execute and Collect)
**函数**: `executeAndCollect(nodes []sqlparser.Statement, statements []string) ([][]byte, error)`

- 接收 SQL AST 节点列表和原始语句列表
- 根据节点类型执行相应的操作：
  - **DQL (SELECT)**: 执行查询，返回 CSV 格式
  - **DML (INSERT/UPDATE/DELETE)**: 执行修改，返回 JSON (last_insert_id, rows_affected)
  - **DDL (CREATE/ALTER/DROP)**: 执行定义，返回 JSON (last_insert_id, rows_affected)
- 收集所有执行结果
- 如果是单条语句，返回原始格式
- 如果是多条语句，返回 batch JSON 格式

**单条语句返回格式**:
- DQL: 直接返回 CSV 字节数组
- DML/DDL: 直接返回 JSON 字节数组

**多条语句返回格式** (Batch):
```json
{
  "batch": true,
  "count": 3,
  "results": [
    {
      "statement": "CREATE TABLE users (id INT)",
      "type": "DDL",
      "data": {"last_insert_id": 0, "rows_affected": 0},
      "error": ""
    },
    {
      "statement": "INSERT INTO users VALUES (1)",
      "type": "DML",
      "data": {"last_insert_id": 1, "rows_affected": 1},
      "error": ""
    },
    {
      "statement": "SELECT * FROM users",
      "type": "DQL",
      "data": "id\n1\n",
      "error": ""
    }
  ]
}
```

---

## 完整调用链

```
用户请求 (SQL string)
    ↓
Execute(sql string)
    ↓
1. splitStatements(sql) → []string
    ↓
2. parseStatements(statements) → []sqlparser.Statement (带日志)
    ↓
3. executeAndCollect(nodes, statements) → [][]byte
    ↓
返回给用户
```

---

## 关键点

1. **默认按 batch 处理**: 即使是单条语句，也先拆分成列表处理
2. **三个独立函数**: 拆分、解析、执行，职责清晰
3. **日志输出**: 在解析步骤输出每条语句的类型
4. **错误处理**: 每条语句独立执行，单条失败不影响其他语句
5. **返回格式**: 单条语句返回原始格式，多条语句返回 batch JSON

---

## Gateway 层职责

Gateway 只负责：
- 接收 HTTP 请求
- 调用 `RDB.Execute(sql)`
- 根据返回结果设置正确的 Content-Type
- 返回响应

**不应该**在 Gateway 层做任何 SQL 解析或判断逻辑。

---

# Manager 架构设计

## 概述

Manager 是一个独立的管理服务，运行在端口 10086，负责管理物理数据库连接（superuser 权限）和逻辑数据库连接（受限权限）。

## 架构层次

```
┌─────────────────────────────────────────────────────────────┐
│                         Manager                              │
│  - HTTP API (port 10086)                                    │
│  - Physical RDB Map (superuser connections)                 │
│  - Physical KV Map                                          │
│  - Config file (JSON format)                                │
└─────────────────────────────────────────────────────────────┘
                    ↓ manages
┌─────────────────────────────────────────────────────────────┐
│                        Processor                             │
│  - Logical RDB Map (limited user connections)               │
│  - Logical KV Map                                           │
│  - Only accesses specific databases with limited privileges │
└─────────────────────────────────────────────────────────────┘
```

## Physical vs Logical 连接

### Physical RDB (物理连接)
- **权限**: Superuser (如 postgres 用户)
- **用途**:
  - 创建数据库
  - 创建用户
  - 授予权限
  - 管理数据库结构
- **管理**: 由 Manager 维护
- **配置**: 在 config.json 的 `physical_rdbs` 中定义

### Logical RDB (逻辑连接)
- **权限**: 受限用户 (如 app_user)
- **用途**:
  - 应用程序日常数据库操作
  - 只能访问特定数据库
  - 不能创建数据库或用户
- **管理**: 由 Processor 维护
- **配置**: 在 config.json 的 `logical_rdbs` 中定义

---

## 配置文件格式

配置文件使用 JSON 格式 (`config.json`)：

```json
{
  "manager": {
    "port": 10086,
    "host": "localhost"
  },
  "physical_rdbs": [
    {
      "id": "psql-main",
      "type": "postgres",
      "host": "localhost",
      "port": 5432,
      "user": "postgres",
      "password": "postgres",
      "dbname": "postgres"
    }
  ],
  "physical_kvs": [],
  "logical_rdbs": [
    {
      "id": "app-db-1",
      "type": "postgres",
      "host": "localhost",
      "port": 5432,
      "user": "app_user",
      "password": "app_pass",
      "dbname": "app_database"
    }
  ],
  "logical_kvs": []
}
```

---

## Manager HTTP API

Manager 提供简单的 HTTP API (端口 10086)，所有操作使用 GET 请求。

### API 端点

#### 1. 健康检查
```
GET /health
```

#### 2. 列出所有 Physical RDBs
```
GET /api/physical/rdbs
```

#### 3. 列出数据库
```
GET /api/physical/rdbs/:id/databases
```

#### 4. 创建 Logical RDB (合并操作)
```
GET /api/logical/rdb/create?rdb_id=<id>&db_name=<name>&username=<user>&password=<pass>
```

这个端点会自动执行：
- 创建数据库
- 创建用户
- 授予权限

---

## CLI 工具

CLI 工具位于 `cmd/cli/main.go`，本质是对 Manager API 的封装调用。

### 安装和使用

```bash
# 构建 CLI
go build -o combinator-cli ./cmd/cli

# 使用 CLI
./combinator-cli <command> [args]
```

### 命令列表

#### 1. 列出所有 Physical RDBs (Manager 管理的)
```bash
combinator-cli manager rdb list
```

#### 2. 列出所有 Logical RDBs (Processor 使用的)
```bash
combinator-cli rdb list
```

#### 3. 列出数据库
```bash
combinator-cli rdb databases <rdb_id>
```

#### 4. 创建 Logical RDB (一次性完成：创建数据库 + 创建用户 + 授予权限)
```bash
combinator-cli create <rdb_id> <db_name> <username> <password>
```

**示例**:
```bash
# 在 psql-main 上创建一个新的应用数据库
combinator-cli create psql-main myapp myapp_user myapp_pass
```

这个命令会自动执行：
1. 创建数据库 `myapp`
2. 创建用户 `myapp_user` (密码: `myapp_pass`)
3. 授予 `myapp_user` 对 `myapp` 数据库的所有权限
