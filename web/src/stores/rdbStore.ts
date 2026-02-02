import { create } from 'zustand'

interface RDBState {
  tables: string[]
  cols: string[]
  rows: string[][]
  loading: boolean
  error: string | null
  setTables: (tables: string[]) => void
  setResult: (cols: string[], rows: string[][]) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  reset: () => void
}

export const useRDBStore = create<RDBState>((set) => ({
  tables: [],
  cols: [],
  rows: [],
  loading: false,
  error: null,
  setTables: (tables) => set({ tables }),
  setResult: (cols, rows) => set({ cols, rows }),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
  reset: () => set({ tables: [], cols: [], rows: [], error: null }),
}))
