import type {
  CombinatorConfig,
  RDBQueryResult,
  RDBOptions,
  KVOptions,
  RDBExecResult,
} from './types'

export class Combinator {
  private baseURL: string

  constructor(config: CombinatorConfig) {
    this.baseURL = config.baseURL.replace(/\/$/, '')
  }

  async request(
    method: string,
    path: string,
    headers?: HeadersInit,
    body?: BodyInit
  ): Promise<Response> {
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

  async exec(statement: string, params?: any[]): Promise<RDBExecResult> {
    const res = await this.combinator.request(
      'POST',
      '/rdb/exec',
      { 'X-Combinator-RDB-ID': this.options.instanceId },
      JSON.stringify({ stmt: statement, args: params || [] })
    )
    if (!res.ok) {
      throw new Error(`RDB exec failed with status ${res.status}`)
    }
    const data = await res.json()
    return data
  }

  async batch(statements: string[]): Promise<void> {
    const res = await this.combinator.request(
      'POST',
      '/rdb/batch',
      { 'X-Combinator-RDB-ID': this.options.instanceId },
      JSON.stringify(statements)
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

  async set(key: string, value: Blob): Promise<void> {
    const res = await this.combinator.request(
      'POST',
      `/kv/set`,
      {
        'Content-Type': 'application/octet-stream',
        'X-Combinator-KV-ID': this.options.instanceId,
        'X-Combinator-KV-Key': key
      },
      value
    )
    if (!res.ok) {
      throw new Error(`KV set failed with status ${res.status}`)
    }
  }
}
