export interface CombinatorConfig {
  baseURL: string
}

export interface RDBOptions {
  instanceId: string
}

export interface RDBQueryOptions {
  sql: string
  params?: any[]
}

export type RDBQueryResult<Item> = {
  columns: string[]
  rows: Item[]
}

export interface RDBExecOptions {
  sql: string
  params?: any[]
}

export interface RDBExecResult {
  rows_affected: number
}

export interface RDBBatchOptions {
  sqls: string[]
}

export interface KVOptions {
  instanceId: string
}

export interface KVGetOptions {
  key: string
}

export interface KVSetOptions {
  key: string
  value: string
}

