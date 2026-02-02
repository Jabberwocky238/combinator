import { useEffect } from 'react'
import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import { useCombinator } from '../context/CombinatorContext'
import { useRDBStore } from '../stores/rdbStore'
import { Backspace20Filled } from '@ricons/fluent'
import { SQLEditor } from './SQLEditor'

const LIST_SQL = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"

export function RDBDetailPanel() {
  const { id = '' } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const { client } = useCombinator()
  const { tables, cols, rows, loading, error, setTables, setResult, setLoading, setError, reset } = useRDBStore()

  const table = searchParams.get('table')
  const rdb = client?.rdb(id)

  const loadTables = async () => {
    if (!rdb) return
    try {
      const res = await rdb.query(LIST_SQL, [])
      setTables(res.rows.map(r => r[0]))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed')
    }
  }

  useEffect(() => {
    reset()
    setLoading(true)
    loadTables().finally(() => setLoading(false))
  }, [client, id])

  const query = async (sql: string) => {
    if (!rdb) return
    setLoading(true)
    setError(null)
    try {
      const res = await rdb.query(sql, [])
      setResult(res.columns || [], res.rows || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Query failed')
      setResult([], [])
    } finally {
      setLoading(false)
    }
  }

  const batch = async (sqls: string[]) => {
    if (!rdb) return
    setLoading(true)
    setError(null)
    try {
      await rdb.batch(sqls, [])
      await loadTables()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Exec failed')
    } finally {
      setLoading(false)
    }
  }

  const selectTable = (name: string) => {
    setSearchParams({ table: name })
    query(`SELECT * FROM "${name}" LIMIT 50`)
  }

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* 左侧表列表 */}
      <div className="w-64 border-r border-zinc-700 bg-zinc-800/50 overflow-y-auto">
        <div className="p-4 border-b border-zinc-700">
          <button
            onClick={() => navigate('/')}
            className="flex items-center gap-2 text-zinc-400 hover:text-white transition-colors"
          >
            <Backspace20Filled />
            <span>Back</span>
          </button>
          <h2 className="mt-3 text-lg font-semibold text-white">{id}</h2>
        </div>

        <div className="p-2">
          <p className="px-2 py-1 text-xs text-zinc-500 uppercase">Tables</p>
          {tables.map(t => (
            <button
              key={t}
              onClick={() => selectTable(t)}
              className={`w-full text-left px-3 py-2 rounded text-sm transition-colors ${
                table === t ? 'bg-blue-600 text-white' : 'text-zinc-300 hover:bg-zinc-700'
              }`}
            >
              {t}
            </button>
          ))}
          {tables.length === 0 && !loading && (
            <p className="px-3 py-2 text-zinc-500 text-sm">No tables found</p>
          )}
        </div>
      </div>

      {/* 右侧数据展示 */}
      <div className="flex-1 overflow-auto p-4">
        <SQLEditor id={id} onQuery={query} onBatch={batch} loading={loading} />

        {error && (
          <div className="mt-4 p-3 bg-red-900/30 border border-red-700 rounded-lg text-red-400">
            {error}
          </div>
        )}

        {loading && (
          <div className="text-zinc-400">Loading...</div>
        )}

        {!loading && table && (
          <div>
            <h3 className="text-lg font-semibold text-white mb-4">
              {table}
              <span className="ml-2 text-sm text-zinc-500 font-normal">({rows.length} rows)</span>
            </h3>

            {cols.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-zinc-700">
                      {cols.map(c => (
                        <th key={c} className="px-3 py-2 text-left text-zinc-400 font-medium">{c}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row, i) => (
                      <tr key={i} className="border-b border-zinc-800 hover:bg-zinc-800/50">
                        {row.map((cell, j) => (
                          <td key={j} className="px-3 py-2 text-zinc-300 font-mono">
                            {cell || <span className="text-zinc-600">NULL</span>}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-zinc-500">No data</p>
            )}
          </div>
        )}

        {!loading && !table && (
          <div className="flex items-center justify-center h-full text-zinc-500">
            Select a table to view data
          </div>
        )}
      </div>
    </div>
  )
}
