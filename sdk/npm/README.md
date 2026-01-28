# Combinator SDK

TypeScript SDK for Combinator - A unified HTTP API gateway for multiple database backends (RDB and KV).

## Installation

```bash
npm install combinator-sdk
# or
yarn add combinator-sdk
# or
pnpm add combinator-sdk
```

## Quick Start

```typescript
import { Combinator } from 'combinator-sdk'

// Initialize client
const combinator = new Combinator({
  baseURL: 'http://localhost:8899'
})

// Get RDB instance
const rdb = combinator.rdb('0')

// Query data
const result = await rdb.query('SELECT * FROM users')
console.log(result.columns, result.rows)
```

## API Reference

### Combinator

Main client class for connecting to Combinator gateway.

```typescript
const combinator = new Combinator({
  baseURL: 'http://localhost:8899'
})
```

#### Methods

- `rdb(instanceId: string): RDB` - Get RDB instance
- `kv(instanceId: string): KV` - Get KV instance

### RDB

Relational database operations.

#### query()

Execute a SELECT query and return results.

```typescript
// Simple query
const result = await rdb.query('SELECT * FROM users')

// Query with parameters
const result = await rdb.query('SELECT * FROM users WHERE id = ?', [1])

// Query with type conversion
interface User {
  id: number
  name: string
  active: boolean
}
const result = await rdb.query<User>(
  'SELECT id, name, active FROM users',
  [],
  ['number', 'string', 'boolean']
)
```

#### exec()

Execute INSERT, UPDATE, DELETE statements.

```typescript
const result = await rdb.exec(
  'INSERT INTO users (name, email) VALUES (?, ?)',
  ['Alice', 'alice@example.com']
)
console.log(result.rows_affected)
```

#### batch()

Execute multiple statements in batch.

```typescript
await rdb.batch([
  'INSERT INTO users (name) VALUES ("User1")',
  'INSERT INTO users (name) VALUES ("User2")'
])
```

### KV

Key-value store operations.

```typescript
const kv = combinator.kv('0')

// Set value
await kv.set('key', new Blob(['value']))

// Get value
const value = await kv.get('key')
```

## License

MIT
