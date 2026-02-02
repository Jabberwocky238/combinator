import { useCallback } from 'react'
import { useCombinator } from '../context/CombinatorContext'

interface JSONRPCRequest {
  jsonrpc: '2.0'
  method: string
  params?: unknown
  id: number | string
}

interface JSONRPCResponse<T = unknown> {
  jsonrpc: '2.0'
  result?: T
  error?: {
    code: number
    message: string
    data?: unknown
  }
  id: number | string
}

let requestId = 0

export function useJSONRPC() {
  const { client, isConnected } = useCombinator()

  const call = useCallback(async <T = unknown>(
    method: string,
    params?: unknown
  ): Promise<T> => {
    if (!client || !isConnected) {
      throw new Error('Not connected')
    }

    const request: JSONRPCRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id: ++requestId
    }

    const res = await client.request('POST', '/monitor', { 
      'Content-Type': 'application/json' 
    }, JSON.stringify(request))

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`)
    }

    const response: JSONRPCResponse<T> = await res.json()

    if (response.error) {
      throw new Error(response.error.message)
    }

    return response.result as T
  }, [client, isConnected])

  return { call, isConnected }
}
