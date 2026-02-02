import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { Combinator } from 'combinator-sdk'

interface CombinatorContextValue {
  endpoint: string | null
  setEndpoint: (endpoint: string) => void
  client: Combinator | null
  isConnected: boolean
  connectionError: string | null
  clearEndpoint: () => void
}

const CombinatorContext = createContext<CombinatorContextValue | null>(null)

const STORAGE_KEY = 'combinator_endpoint'

export function CombinatorProvider({ children }: { children: ReactNode }) {
  const [endpoint, setEndpointState] = useState<string | null>(() => {
    return localStorage.getItem(STORAGE_KEY)
  })
  const [client, setClient] = useState<Combinator | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [connectionError, setConnectionError] = useState<string | null>(null)

  const setEndpoint = (newEndpoint: string) => {
    localStorage.setItem(STORAGE_KEY, newEndpoint)
    setEndpointState(newEndpoint)
  }

  const clearEndpoint = () => {
    localStorage.removeItem(STORAGE_KEY)
    setEndpointState(null)
    setClient(null)
    setIsConnected(false)
  }

  useEffect(() => {
    if (!endpoint) {
      setClient(null)
      setIsConnected(false)
      setConnectionError(null)
      return
    }

    const newClient = new Combinator({ baseURL: endpoint })
    setClient(newClient)

    newClient.request('GET', '/health')
      .then(res => {
        if (res.ok) {
          setIsConnected(true)
          setConnectionError(null)
        } else {
          setIsConnected(false)
          setConnectionError(`HTTP ${res.status}`)
        }
      })
      .catch(err => {
        setIsConnected(false)
        setConnectionError(err.message)
      })
  }, [endpoint])

  return (
    <CombinatorContext.Provider value={{
      endpoint,
      setEndpoint,
      client,
      isConnected,
      connectionError,
      clearEndpoint
    }}>
      {children}
    </CombinatorContext.Provider>
  )
}

export function useCombinator() {
  const ctx = useContext(CombinatorContext)
  if (!ctx) throw new Error('useCombinator must be used within CombinatorProvider')
  return ctx
}
