import { useState, useEffect, useCallback } from 'react'
import { useJSONRPC } from './useJSONRPC'

export interface ServiceInfo {
  id: string
  type: string
}

export interface ServiceList {
  rdb: ServiceInfo[]
  kv: ServiceInfo[]
}

export function useServiceList() {
  const { call, isConnected } = useJSONRPC()
  const [services, setServices] = useState<ServiceList>({ rdb: [], kv: [] })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!isConnected) return

    setLoading(true)
    setError(null)

    try {
      const result = await call<ServiceList>('service.list')
      setServices({
        rdb: result.rdb || [],
        kv: result.kv || []
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [call, isConnected])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { services, loading, error, refresh }
}
