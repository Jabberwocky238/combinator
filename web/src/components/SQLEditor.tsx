import { useState, useEffect } from 'react'

interface Props {
  id: string
  onQuery: (sql: string) => void
  onBatch: (sqls: string[]) => void
  loading: boolean
}

export function SQLEditor({ id, onQuery, onBatch, loading }: Props) {
  const key = `sql_${id}`
  const [exec, setExec] = useState(`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO users (name) VALUES ('Alice'), ('Bob'), ('Charlie');
  `)
  const [query, setQuery] = useState('* FROM users LIMIT 50')

  useEffect(() => {
    const saved = localStorage.getItem(key)
    if (saved) setExec(saved)
  }, [key])

  const updateExec = (v: string) => {
    setExec(v)
    localStorage.setItem(key, v)
  }

  const runQuery = () => query.trim() && onQuery(`SELECT ${query.trim()}`)
  const runBatch = () => exec.trim() && onBatch(exec.trim().split(';').map(s => s.trim()).filter(s => s))

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <span className="text-zinc-400 font-mono text-sm">SELECT</span>
        <input
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && runQuery()}
          placeholder="* FROM users LIMIT 50"
          className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded text-zinc-100 font-mono text-sm focus:outline-none focus:border-blue-500"
        />
        <button
          onClick={runQuery}
          disabled={loading || !query.trim()}
          className="px-3 py-2 bg-green-600 hover:bg-green-700 disabled:bg-zinc-700 text-white text-sm rounded transition-colors disabled:cursor-not-allowed"
        >
          Query
        </button>
      </div>

      <div className="border border-zinc-700 rounded-lg overflow-hidden">
        <div className="flex items-center justify-between px-3 py-2 bg-zinc-800 border-b border-zinc-700">
          <span className="text-sm text-zinc-400">Exec (CREATE / INSERT / UPDATE / DELETE)</span>
          <button
            onClick={runBatch}
            disabled={loading || !exec.trim()}
            className="px-3 py-1 bg-orange-600 hover:bg-orange-700 disabled:bg-zinc-700 text-white text-sm rounded transition-colors disabled:cursor-not-allowed"
          >
            {loading ? 'Running...' : 'Exec'}
          </button>
        </div>
        <textarea
          value={exec}
          onChange={e => updateExec(e.target.value)}
          onKeyDown={e => (e.ctrlKey || e.metaKey) && e.key === 'Enter' && runBatch()}
          placeholder="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"
          className="w-full h-32 p-3 bg-zinc-900 text-zinc-100 font-mono text-sm resize-y focus:outline-none placeholder-zinc-600"
          spellCheck={false}
        />
      </div>
    </div>
  )
}
