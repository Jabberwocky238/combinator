#!/bin/bash

BASE_URL="http://localhost:8899/rdb"

create_table() {
    local rdb_id=$1
    sql='CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        content TEXT
    )'

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

insert_data() {
    local rdb_id=$1
    sql="INSERT INTO users (name, content) VALUES ('test_user', 'test_content')"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

select_data() {
    local rdb_id=$1
    sql="SELECT * FROM users"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

batch_dml() {
    local rdb_id=$1
    sql="\
INSERT INTO users (name, content) VALUES ('user1', 'content1');
INSERT INTO users (name, content) VALUES ('user2', 'content2');
"
    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

batch_ddl() {
    local rdb_id=$1

    # 根据数据库类型使用不同的 SQL
    if [ "$rdb_id" = "1" ]; then
        # SQLite
        sql1="\
CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    price REAL
);"
    else
        # PostgreSQL
        sql1="\
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price REAL
);"
    fi
    sql2="\
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);

INSERT INTO products (name, price) VALUES ('product1', 10.0);
INSERT INTO products (name, price) VALUES ('product2', 20.0);
"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql1$sql2" \
        -w "\nHTTP Status: %{http_code}\n"
}

error_batch() {
    local rdb_id=$1
    sql="\
CREATE TABLE IF NOT EXISTS users111 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    content TEXT
);

INSERT INTO users111 (name, content) VALUES ('user1', 'content3');
INSERT INTO users111 (name, content) VALUES ('user1', 'content_duplicate');
"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

error_batch2_res() {
    local rdb_id=$1
    sql="\
SELECT * FROM user111
"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

DQL_with_DML() {
    local rdb_id=$1

    # 根据数据库类型使用不同的 SQL
    if [ "$rdb_id" = "1" ]; then
        # SQLite
        sql="\
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    content TEXT
);
INSERT INTO users (name, content) VALUES ('user1', 'content1');
INSERT INTO users (name, content) VALUES ('user2', 'content2');
INSERT INTO users (name, content) VALUES ('user3', 'content3');
SELECT * FROM users;
"
    else
        # PostgreSQL
        sql="\
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    content TEXT
);
INSERT INTO users (name, content) VALUES ('user1', 'content1');
INSERT INTO users (name, content) VALUES ('user2', 'content2');
INSERT INTO users (name, content) VALUES ('user3', 'content3');
SELECT * FROM users;
"
    fi

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: $rdb_id" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

# 主逻辑
if [ $# -lt 2 ]; then
    echo "Usage: bash fetch.sh <db_id> <function_id>"
    echo "  db_id: Database ID (1 for SQLite, 2 for PostgreSQL)"
    echo "  function_id:"
    echo "    1 - Create table"
    echo "    2 - Insert data"
    echo "    3 - Select data"
    echo "    4 - Batch DML operations"
    echo "    5 - Batch DDL operations"
    echo "    6 - Error handling in batch DML"
    echo "    7 - DQL with DML"
    echo "    8 - after 6"
    echo "Example: bash fetch.sh 1 3  (run select_data on SQLite)"
    exit 1
fi

DB_ID=$1
FUNC_ID=$2

case "$FUNC_ID" in
    1)
        create_table "$DB_ID"
        ;;
    2)
        insert_data "$DB_ID"
        ;;
    3)
        select_data "$DB_ID"
        ;;
    4)
        batch_dml "$DB_ID"
        ;;
    5)
        batch_ddl "$DB_ID"
        ;;
    6)
        error_batch "$DB_ID"
        ;;
    7)
        DQL_with_DML "$DB_ID"
        ;;
    8)
        error_batch2_res "$DB_ID"
        ;;
    *)
        echo "Invalid function choice: $FUNC_ID"
        echo "Please use 1-8"
        exit 1
        ;;
esac
