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

export interface KVOptions {
  instanceId: string
}



