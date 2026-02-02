import { useState, useEffect } from 'react'
import { useCombinator } from '../context/CombinatorContext'
import { Backspace20Filled } from '@ricons/fluent'
import { SQLEditor } from './SQLEditor'

interface TableInfo {
  name: string
}

interface RDBDetailPanelProps {
  serviceId: string
  serviceType: string
  onBack: () => void
}

export function RDBDetailPanel({ serviceId, serviceType, onBack }: RDBDetailPanelProps) {
  const { client } = useCombinator()
  const [tables, setTables] = useState<TableInfo[]>([])
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [tableData, setTableData] = useState<string[][]>([])
  const [columns, setColumns] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // 获取表列表
  useEffect(() => {
    if (!client) return

    const fetchTables = async () => {
      setLoading(true)
      try {
        const sql = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
        const res = await client.rdb(serviceId).query(sql, [])

        if (res.rows.length > 1) {
          setTables(res.rows.slice(1).map(row => ({ name: row[0] })))
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch tables')
      } finally {
        setLoading(false)
      }
    }

    fetchTables()
  }, [client, serviceId, serviceType])

  // 查询表数据
  const queryTable = async (tableName: string) => {
    if (!client) return

    setSelectedTable(tableName)
    setLoading(true)
    setError(null)

    try {
      const sql = `SELECT * FROM "${tableName}" LIMIT 50`
      const res = await client.rdb(serviceId).query(sql, [])

      if (res.rows.length > 0) {
        setColumns(res.columns)
        setTableData(res.rows)
      } else {
        setColumns([])
        setTableData([])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Query failed')
    } finally {
      setLoading(false)
    }
  }

  // 执行自定义 SQL (SELECT)
  const executeSQL = async (sql: string) => {
    if (!client) return

    setSelectedTable(null)
    setLoading(true)
    setError(null)

    try {
      const res = await client.rdb(serviceId).query(sql, [])
      setColumns(res.columns || [])
      setTableData(res.rows || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Query failed')
      setColumns([])
      setTableData([])
    } finally {
      setLoading(false)
    }
  }

  // 执行 exec (CREATE/INSERT/UPDATE/DELETE)
  const execSQL = async (sql: string) => {
    if (!client) return

    setLoading(true)
    setError(null)

    try {
      await client.rdb(serviceId).exec(sql, [])
      // 刷新表列表
      const listSql = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
      const res = await client.rdb(serviceId).query(listSql, [])
      if (res.rows.length > 0) {
        setTables(res.rows.map(row => ({ name: row[0] })))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Exec failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      {/* 左侧表列表 */}
      <div className="w-64 border-r border-zinc-700 bg-zinc-800/50 overflow-y-auto">
        <div className="p-4 border-b border-zinc-700">
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-zinc-400 hover:text-white transition-colors"
          >
            <Backspace20Filled />
            <span>Back</span>
          </button>
          <h2 className="mt-3 text-lg font-semibold text-white">{serviceId}</h2>
          <p className="text-xs text-zinc-500 uppercase">{serviceType}</p>
        </div>

        <div className="p-2">
          <p className="px-2 py-1 text-xs text-zinc-500 uppercase">Tables</p>
          {tables.map(table => (
            <button
              key={table.name}
              onClick={() => queryTable(table.name)}
              className={`w-full text-left px-3 py-2 rounded text-sm transition-colors ${
                selectedTable === table.name
                  ? 'bg-blue-600 text-white'
                  : 'text-zinc-300 hover:bg-zinc-700'
              }`}
            >
              {table.name}
            </button>
          ))}
          {tables.length === 0 && !loading && (
            <p className="px-3 py-2 text-zinc-500 text-sm">No tables found</p>
          )}
        </div>
      </div>

      {/* 右侧数据展示 */}
      <div className="flex-1 overflow-auto p-4">
        <SQLEditor serviceId={serviceId} onQuery={executeSQL} onExec={execSQL} loading={loading} />

        {error && (
          <div className="mt-4 p-3 bg-red-900/30 border border-red-700 rounded-lg text-red-400">
            {error}
          </div>
        )}

        {loading && (
          <div className="text-zinc-400">Loading...</div>
        )}

        {!loading && selectedTable && (
          <div>
            <h3 className="text-lg font-semibold text-white mb-4">
              {selectedTable}
              <span className="ml-2 text-sm text-zinc-500 font-normal">
                ({tableData.length} rows)
              </span>
            </h3>

            {columns.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-zinc-700">
                      {columns.map(col => (
                        <th key={col} className="px-3 py-2 text-left text-zinc-400 font-medium">
                          {col}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {tableData.map((row, i) => (
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

        {!loading && !selectedTable && (
          <div className="flex items-center justify-center h-full text-zinc-500">
            Select a table to view data
          </div>
        )}
      </div>
    </div>
  )
}
