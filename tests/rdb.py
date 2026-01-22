import requests

headers = {
    "X-Combinator-RDB-ID": "1"
}

def test_batch_endpoint():
    url = f"http://localhost:8899/rdb/batch"
    payload = [
        """
        CREATE TABLE IF NOT EXISTS test_batch (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL
        );
        """,
        """
        INSERT INTO test_batch (name) VALUES ('Alice');
        """,
        "INSERT INTO test_batch (name) VALUES ('Charlie');"
    ]
    response = requests.post(url, json=payload, headers=headers)
    assert response.status_code == 200
    
def test_exec_endpoint():
    url = f"http://localhost:8899/rdb/exec"
    payload = {
        "stmt": "INSERT INTO test_batch (name) VALUES (?);",
        "args": ["Bob"]
    }
    response = requests.post(url, json=payload, headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert 'rows_affected' in data
    
def test_query_endpoint():
    url = f"http://localhost:8899/rdb/query"
    payload = {
        "stmt": "SELECT * FROM test_batch",
        "args": []
    }
    response = requests.post(url, json=payload, headers=headers)
    assert response.status_code == 200
    assert response.headers['Content-Type'] == 'application/csv'
    assert response.text.count('\n') >= 4  # At least header + 3 rows
    print(response.text)
    
def test_complicated_migration():
    url = f"http://localhost:8899/rdb/batch"
    payload = [
        # Step 1: 创建新表 test_batch2，包含 description 字段
        """
        CREATE TABLE IF NOT EXISTS test_batch2 (
            id INTEGER PRIMARY KEY,
            description TEXT NOT NULL
        );
        """,

        # Step 2: 将旧表 test_batch 的 name 数据迁移到新表的 description 字段
        """
        INSERT INTO test_batch2 (description)
        SELECT name FROM test_batch;
        """,

        # Step 3: 删除旧表
        """
        DROP TABLE test_batch;
        """,

        # Step 4: 将新表重命名为旧表名
        """
        ALTER TABLE test_batch2 RENAME TO test_batch;
        """,
        "ALTER TABLE test_batch RENAME COLUMN description TO name;"
    ]

    response = requests.post(url, json=payload, headers=headers)
    assert response.status_code == 200
    print("Migration completed successfully!")

    # 验证迁移结果
    verify_url = f"http://localhost:8899/rdb/query"
    verify_payload = {
        "stmt": "SELECT * FROM test_batch",
        "args": []
    }
    verify_response = requests.post(verify_url, json=verify_payload, headers=headers)
    assert verify_response.status_code == 200
    print("Verification query result:")
    print(verify_response.text)
    
if __name__ == "__main__":
    test_batch_endpoint()
    test_exec_endpoint()
    test_query_endpoint()
    test_complicated_migration()