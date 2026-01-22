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
    assert 'last_insert_id' in data
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
    
if __name__ == "__main__":
    test_batch_endpoint()
    test_exec_endpoint()
    test_query_endpoint()