import { describe, it, expect } from 'vitest'
import { Combinator } from '../src/client'

const BASE_URL = 'http://localhost:8899'
const RDB_ID = '0'

const combinator = new Combinator({ baseURL: BASE_URL })
const rdb = combinator.rdb(RDB_ID)

describe('RDB Complete Flow Test', () => {
  it('should complete full CRUD flow', async () => {
    // Step 1: Create table with exec
    await rdb.exec('CREATE TABLE IF NOT EXISTS test_users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)')

    // Step 2: Insert data with exec
    const insertResult = await rdb.exec(
      'INSERT INTO test_users (name, email) VALUES (?, ?)',
      ['Alice', 'alice@test.com']
    )
    console.log('insertResult:', insertResult)
    expect(insertResult.rows_affected).toBe(1)

    // Step 3: Query data
    const queryResult = await rdb.query('SELECT * FROM test_users WHERE name = ?', ['Alice'])
    expect(queryResult.columns).toContain('name')
    expect(queryResult.rows.length).toBeGreaterThan(0)

    // Step 4: Batch operations
    await rdb.batch([
      "INSERT INTO test_users (name, email) VALUES ('Bob', 'bob@test.com');",
      "INSERT INTO test_users (name, email) VALUES ('Charlie', 'charlie@test.com');"
    ])

    // Step 5: Verify batch insert
    const allUsers = await rdb.query('SELECT * FROM test_users')
    expect(allUsers.rows.length).toBeGreaterThanOrEqual(3)
  })
})
