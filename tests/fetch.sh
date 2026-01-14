#!/bin/bash

BASE_URL="http://localhost:8899/rdb"

create_table() {
    sql='CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        content TEXT
    )'

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n" 
}

insert_data() {
    sql="INSERT INTO users (name, content) VALUES ('test_user', 'test_content')"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

select_data() {
    sql="SELECT * FROM users"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

batch_dml() {
    sql="\
INSERT INTO users (name, content) VALUES ('user1', 'content1');
INSERT INTO users (name, content) VALUES ('user2', 'content2');
"
    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

batch_ddl() {
    sql="\
CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    price REAL
);

CREATE INDEX idx_products_name ON products(name);

INSERT INTO products (name, price) VALUES ('product1', 10.0);
INSERT INTO products (name, price) VALUES ('product2', 20.0);
"
    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

error_batch() {
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
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

error_batch2_res() {
    sql="\
SELECT * FROM user111
"

    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

DQL_with_DML() {
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
    curl -X POST "$BASE_URL" \
        -H "Content-Type: application/sql" \
        -H "X-Combinator-RDB-ID: 1" \
        -d "$sql" \
        -w "\nHTTP Status: %{http_code}\n"
}

# 主逻辑
if [ $# -eq 0 ]; then
    echo "Usage: bash fetch.sh [1|2|3]"
    echo "  1 - Create table"
    echo "  2 - Insert data"
    echo "  3 - Select data"
    echo "  4 - Batch DML operations"
    echo "  5 - Batch DDL operations"
    echo "  6 - Error handling in batch DML"
    echo "  7 - DQL with DML"
    echo "  8 - after 6"
    echo "Please provide one of the above options."
    exit 1
fi

case "$1" in
    1)
        create_table
        ;;
    2)
        insert_data
        ;;
    3)
        select_data
        ;;
    4)
        batch_dml
        ;;
    5)
        batch_ddl
        ;;
    6)
        error_batch
        ;;
    7)
        DQL_with_DML
        ;;
    8)
        error_batch2_res
        ;;
    *)
        echo "Invalid choice: $1"
        echo "Please use 1, 2, or 3"
        exit 1
        ;;
esac
