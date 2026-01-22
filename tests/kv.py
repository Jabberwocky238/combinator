#!/usr/bin/env python3
"""
KV Gateway API Tests
Tests the KV get/set operations via HTTP API
"""

import requests
import json

# Configuration
BASE_URL = "http://localhost:8899"
KV_ID = "1"  # Change this to match your config


def test_set_get():
    """Test basic set and get operations"""
    print("Testing SET and GET operations...")

    # Set a value
    key = "test-key-1"
    value = b"Hello, World!"

    set_response = requests.post(
        f"{BASE_URL}/kv/set",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        },
        data=value
    )

    print(f"SET Response: {set_response.status_code} - {set_response.text}")
    assert set_response.status_code == 200, f"SET failed: {set_response.text}"

    # Get the value
    get_response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        }
    )

    print(f"GET Response: {get_response.status_code}")
    print(f"GET Value: {get_response.content}")
    assert get_response.status_code == 200, f"GET failed: {get_response.text}"
    assert get_response.content == value, f"Value mismatch: {get_response.content} != {value}"

    print("✓ SET and GET test passed\n")


def test_binary_data():
    """Test storing and retrieving binary data"""
    print("Testing binary data...")

    key = "test-binary"
    value = bytes(range(256))  # All byte values 0-255

    # Set binary data
    set_response = requests.post(
        f"{BASE_URL}/kv/set",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        },
        data=value
    )

    assert set_response.status_code == 200

    # Get binary data
    get_response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        }
    )

    assert get_response.status_code == 200
    assert get_response.content == value

    print("✓ Binary data test passed\n")


def test_overwrite():
    """Test overwriting existing key"""
    print("Testing overwrite...")

    key = "test-overwrite"
    value1 = b"First value"
    value2 = b"Second value"

    # Set first value
    requests.post(
        f"{BASE_URL}/kv/set",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        },
        data=value1
    )

    # Set second value (overwrite)
    requests.post(
        f"{BASE_URL}/kv/set",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        },
        data=value2
    )

    # Get value
    get_response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        }
    )

    assert get_response.content == value2, "Overwrite failed"

    print("✓ Overwrite test passed\n")


def test_missing_key():
    """Test getting a non-existent key"""
    print("Testing missing key...")

    key = "non-existent-key-12345"

    get_response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        }
    )

    print(f"Missing key response: {get_response.status_code}")
    assert get_response.status_code == 500, "Should return error for missing key"

    print("✓ Missing key test passed\n")


def test_missing_headers():
    """Test requests with missing headers"""
    print("Testing missing headers...")

    # Missing KV-ID
    response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-Key": "test",
        }
    )
    assert response.status_code == 400
    print(f"Missing KV-ID: {response.json()}")

    # Missing KV-Key
    response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
        }
    )
    assert response.status_code == 400
    print(f"Missing KV-Key: {response.json()}")

    print("✓ Missing headers test passed\n")


def test_large_value():
    """Test storing large values"""
    print("Testing large value (1MB)...")

    key = "test-large"
    value = b"x" * (1024 * 1024)  # 1MB

    set_response = requests.post(
        f"{BASE_URL}/kv/set",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        },
        data=value
    )

    assert set_response.status_code == 200

    get_response = requests.get(
        f"{BASE_URL}/kv/get",
        headers={
            "X-Combinator-KV-ID": KV_ID,
            "X-Combinator-KV-Key": key,
        }
    )

    assert get_response.status_code == 200
    assert len(get_response.content) == len(value)

    print("✓ Large value test passed\n")


if __name__ == "__main__":
    print("=" * 50)
    print("KV Gateway API Tests")
    print("=" * 50)
    print(f"Base URL: {BASE_URL}")
    print(f"KV ID: {KV_ID}")
    print("=" * 50)
    print()

    try:
        test_set_get()
        test_binary_data()
        test_overwrite()
        test_missing_key()
        test_missing_headers()
        test_large_value()

        print("=" * 50)
        print("All tests passed! ✓")
        print("=" * 50)
    except AssertionError as e:
        print(f"\n✗ Test failed: {e}")
        exit(1)
    except requests.exceptions.ConnectionError:
        print(f"\n✗ Cannot connect to {BASE_URL}")
        print("Make sure the server is running with: make dev")
        exit(1)
