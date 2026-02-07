export interface CombinatorConfig {
  baseURL?: string
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

export interface KVSetOptions {
  ttl?: string  // 例如: "10s", "1h"
  nx?: boolean
  xx?: boolean
}

export interface KVDelOptions {
  cas?: Blob
}

export interface S3Options {
  instanceId: string
}

export interface S3ObjectInfo {
  key: string
  size: number
  lastModified: string
  etag?: string
  contentType?: string
  metadata?: Record<string, string>
}

export interface S3GetOptions {
  range?: {
    start: number
    end: number
  }
}

export interface S3PutOptions {
  contentType?: string
  metadata?: Record<string, string>
}

export type S3DeleteMode = 'precise' | 'prefix'

export interface S3DeleteKey {
  mode: S3DeleteMode
  key: string
}

export interface S3DeleteOptions {
  keys: S3DeleteKey[]
}

export interface S3DeleteResult {
  deleted: number
}

export interface S3ListOptions {
  prefix?: string
  maxKeys?: number
  marker?: string
}

export interface S3ListResult {
  objects: S3ObjectInfo[]
  isTruncated: boolean
  nextMarker?: string
}
