import type {
  CombinatorConfig,
  RDBQueryResult,
  RDBOptions,
  KVOptions,
  KVSetOptions,
  KVDelOptions,
  S3Options,
  S3ObjectInfo,
  S3GetOptions,
  S3PutOptions,
  S3DeleteOptions,
  S3DeleteResult,
  S3ListOptions,
  S3ListResult,
} from './types'
import { HMACSignature } from './crypto'

export class Combinator {
  isLocal = false
  private baseURL: string
  private uid?: string
  private secretKey?: string

  constructor(config: CombinatorConfig = {}) {
    this.baseURL = (config.baseURL ?? process.env.COMBINATOR_BASE_URL ?? 'http://localhost:8899').replace(/\/$/, '')
    if (this.baseURL.includes('localhost') && this.baseURL.startsWith('http://') && !this.baseURL.startsWith('https://')) {
      this.isLocal = true
    }
    this.uid = config.uid ?? process.env.RAYSAIL_UID
    this.secretKey = config.secretKey ?? process.env.RAYSAIL_SECRET_KEY
  }

  async request(
    method: string,
    path: string,
    headers?: HeadersInit,
    body?: BodyInit
  ): Promise<Response> {
    if (!this.isLocal) {
      if (!this.uid || !this.secretKey) {
        throw new Error('UID and Secret Key are required for remote requests')
      }

      // Sign the request body (same as Go implementation)
      let bodyData: Uint8Array
      if (body instanceof Uint8Array) {
        bodyData = body
      } else if (typeof body === 'string') {
        bodyData = new TextEncoder().encode(body)
      } else if (body instanceof Blob) {
        bodyData = new Uint8Array(await body.arrayBuffer())
      } else if (body) {
        bodyData = new TextEncoder().encode(JSON.stringify(body))
      } else {
        bodyData = new Uint8Array(0)
      }

      const signature = await HMACSignature.generate(this.secretKey, bodyData)
      headers = {
        ...headers,
        'X-Raysail-UID': this.uid,
        'X-Raysail-Signature': signature,
      }
    }
    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers: headers,
      body: body,
    })
    return response
  }

  rdb(id: string): RDB {
    return new RDB(this, { instanceId: id })
  }

  kv(id: string): KV {
    return new KV(this, { instanceId: id })
  }

  s3(id: string): S3 {
    return new S3(this, { instanceId: id })
  }
}

export class RDB {
  private combinator: Combinator
  private options: RDBOptions

  constructor(combinator: Combinator, options: RDBOptions) {
    this.combinator = combinator
    this.options = options
  }

  async query<Item = any>(statement: string, params: any[] = [], schemaType?: string[]): Promise<RDBQueryResult<Item>> {
    if (schemaType) {
      // could be string, number, boolean, any
      const isValidSchema = schemaType.map((type) =>
        ['string', 'number', 'boolean'].includes(type)
      ).every((v) => v)
      if (!isValidSchema) {
        throw new Error('Invalid schemaType provided')
      }
    }

    const res = await this.combinator.request(
      'POST',
      '/rdb/query',
      { 'X-Combinator-RDB-ID': this.options.instanceId },
      JSON.stringify({ stmt: statement, args: params })
    )
    if (!res.ok) {
      throw new Error(`RDB query failed with status ${res.status}`)
    }
    // data should be CSV format
    const data = await res.text()
    const lines = data.trim().split('\n')
    const columns = lines[0].split(',')
    const rows = lines.slice(1).map((line) => line.split(','))
    if (schemaType) {
      const parsedRows = this.parseQueryResult<Item>(rows, schemaType)
      return { columns, rows: parsedRows }
    }
    return { columns, rows } as RDBQueryResult<Item>
  }

  private parseQueryResult<Item = any>(rows: any[][], schemaTypes: string[]): Item[] {
    // perform basic type conversion based on schemaType
    return rows.map((row) =>
      row.map((value, index) => {
        const type = schemaTypes[index]
        if (type === 'number') {
          return parseFloat(value)
        } else if (type === 'boolean') {
          return value === 'true'
        } else if (type === 'string') {
          return `${value}`
        } else {
          return value
        }
      }) as unknown as Item
    )
  }

  async exec(statement: string, params: any[] = []): Promise<void> {
    const res = await this.combinator.request(
      'POST',
      '/rdb/exec',
      { 'X-Combinator-RDB-ID': this.options.instanceId },
      JSON.stringify({ stmt: statement, args: params })
    )
    if (!res.ok) {
      throw new Error(`RDB exec failed with status ${res.status}`)
    }
  }

  async batch(statements: string[], paramsArray: any[][] = []): Promise<void> {
    const res = await this.combinator.request(
      'POST',
      '/rdb/batch',
      { 'X-Combinator-RDB-ID': this.options.instanceId },
      JSON.stringify([...statements.map((stmt, index) => ({ stmt, args: paramsArray[index] || [] }))])
    )
    if (!res.ok) {
      throw new Error(`RDB batch failed with status ${res.status}`)
    }
  }
}

export class KV {
  private combinator: Combinator
  private options: KVOptions

  constructor(combinator: Combinator, options: KVOptions) {
    this.combinator = combinator
    this.options = options
  }

  async get(key: string): Promise<Blob | null> {
    const res = await this.combinator.request(
      'GET',
      `/kv/get`,
      {
        'Content-Type': 'application/octet-stream',
        'X-Combinator-KV-ID': this.options.instanceId,
        'X-Combinator-KV-Key': key
      }
    )
    if (!res.ok) {
      throw new Error(`KV get failed with status ${res.status}`)
    }
    const data = await res.blob()
    return data
  }

  async set(key: string, value: Blob | File, options?: KVSetOptions): Promise<void> {
    const headers: HeadersInit = {
      'X-Combinator-KV-ID': this.options.instanceId,
      'X-Combinator-KV-Key': key
    }

    // 添加 options 到请求头
    if (options) {
      headers['X-Combinator-KV-Options'] = JSON.stringify(options)
    }

    const res = await this.combinator.request(
      'POST',
      `/kv/set`,
      headers,
      value
    )
    if (!res.ok) {
      throw new Error(`KV set failed with status ${res.status}`)
    }
  }

  async del(key: string, options?: KVDelOptions): Promise<void> {
    const headers: HeadersInit = {
      'X-Combinator-KV-ID': this.options.instanceId,
      'X-Combinator-KV-Key': key
    }

    // 添加 options 到请求头
    if (options) {
      headers['X-Combinator-KV-Options'] = JSON.stringify({
        cas: !!(options.cas)
      })
    }

    // 如果是 CAS 删除，需要传递 value
    let body: BodyInit | undefined
    if (options?.cas) {
      body = options.cas
    }

    const res = await this.combinator.request(
      'POST',
      `/kv/del`,
      headers,
      body
    )
    if (!res.ok) {
      throw new Error(`KV delete failed with status ${res.status}`)
    }
  }
}

export class S3 {
  private combinator: Combinator
  private options: S3Options

  constructor(combinator: Combinator, options: S3Options) {
    this.combinator = combinator
    this.options = options
  }

  async head(key: string): Promise<S3ObjectInfo> {
    const res = await this.combinator.request(
      'POST',
      '/s3/head',
      {
        'X-Combinator-S3-ID': this.options.instanceId,
        'X-Combinator-S3-Object-Key': key
      }
    )
    if (!res.ok) {
      throw new Error(`S3 head failed with status ${res.status}`)
    }
    return await res.json()
  }

  async get(key: string, options?: S3GetOptions): Promise<Blob> {
    const headers: HeadersInit = {
      'X-Combinator-S3-ID': this.options.instanceId,
      'X-Combinator-S3-Object-Key': key
    }

    if (options?.range) {
      headers['Range'] = `bytes=${options.range.start}-${options.range.end}`
    }

    const res = await this.combinator.request(
      'POST',
      '/s3/get',
      headers
    )
    if (!res.ok) {
      throw new Error(`S3 get failed with status ${res.status}`)
    }
    return await res.blob()
  }

  async put(key: string, data: Blob | File, options?: S3PutOptions): Promise<void> {
    const headers: HeadersInit = {
      'X-Combinator-S3-ID': this.options.instanceId,
      'X-Combinator-S3-Object-Key': key
    }

    if (options?.contentType) {
      headers['Content-Type'] = options.contentType
    }

    const res = await this.combinator.request(
      'POST',
      '/s3/put',
      headers,
      data
    )
    if (!res.ok) {
      throw new Error(`S3 put failed with status ${res.status}`)
    }
  }

  async delete(options: S3DeleteOptions): Promise<S3DeleteResult> {
    const res = await this.combinator.request(
      'POST',
      '/s3/delete',
      {
        'X-Combinator-S3-ID': this.options.instanceId,
        'Content-Type': 'application/json'
      },
      JSON.stringify(options)
    )
    if (!res.ok) {
      throw new Error(`S3 delete failed with status ${res.status}`)
    }
    return await res.json()
  }

  async copy(srcKey: string, dstKey: string): Promise<void> {
    const res = await this.combinator.request(
      'POST',
      '/s3/copy',
      {
        'X-Combinator-S3-ID': this.options.instanceId,
        'Content-Type': 'application/json'
      },
      JSON.stringify({ src_key: srcKey, dst_key: dstKey })
    )
    if (!res.ok) {
      throw new Error(`S3 copy failed with status ${res.status}`)
    }
  }

  async list(options?: S3ListOptions): Promise<S3ListResult> {
    const res = await this.combinator.request(
      'POST',
      '/s3/list',
      {
        'X-Combinator-S3-ID': this.options.instanceId,
        'Content-Type': 'application/json'
      },
      JSON.stringify(options || {})
    )
    if (!res.ok) {
      throw new Error(`S3 list failed with status ${res.status}`)
    }
    return await res.json()
  }
}