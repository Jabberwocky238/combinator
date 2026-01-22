import requests


rdb_setup = {
    "rdb_id": "1"
}

def test_batch_endpoint(rdb_setup):
    rdb_id = rdb_setup['rdb_id']
    url = f"http://localhost:8899/rdb/{rdb_id}/batch"
    payload = {
        "batch": [
            { 
                "stmt": """
CREATE TABLE IF NOT EXISTS test (
 id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
    
            }
"""
            }
        ]
    }
    response = requests.post(url, json=payload)
    assert response.status_code == 200
    data = response.json()
    assert "results" in data
    assert len(data["results"]) == 3
    assert data["results"][0]["rows_affected"] == 1
    assert data["results"][1]["rows_affected"] == 1
    assert data["results"][2]["rows_affected"] == 1