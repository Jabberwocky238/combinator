import { useState, useEffect } from 'react'

interface SQLEditorProps {
  serviceId: string
  onQuery: (sql: string) => void
  onExec: (sql: string) => void
  loading: boolean
}

const STORAGE_KEY_PREFIX = 'combinator_sql_'

export function SQLEditor({ serviceId, onQuery, onExec, loading }: SQLEditorProps) {
  const storageKey = `${STORAGE_KEY_PREFIX}${serviceId}`
  const [sql, setSql] = useState('')
  const [selectInput, setSelectInput] = useState('')

  // 从 localStorage 加载
  useEffect(() => {
    const saved = localStorage.getItem(storageKey)
    if (saved) setSql(saved)
  }, [storageKey])

  // 保存到 localStorage
  const handleChange = (value: string) => {
    setSql(value)
    localStorage.setItem(storageKey, value)
  }

  const handleExecute = () => {
    if (sql.trim()) {
      onExec(sql.trim())
    }
  }

  const handleSelect = () => {
    if (selectInput.trim()) {
      onQuery(`SELECT ${selectInput.trim()}`)
    }
  }

  const handleSelectKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleSelect()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      handleExecute()
    }
  }

  return (
    <div className="space-y-3">
      {/* SELECT 快捷输入 */}
      <div className="flex items-center gap-2">
        <span className="text-zinc-400 font-mono text-sm">SELECT</span>
        <input
          type="text"
          value={selectInput}
          onChange={e => setSelectInput(e.target.value)}
          onKeyDown={handleSelectKeyDown}
          placeholder="* FROM users LIMIT 50"
          className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded
                     text-zinc-100 font-mono text-sm focus:outline-none focus:border-blue-500"
        />
        <button
          onClick={handleSelect}
          disabled={loading || !selectInput.trim()}
          className="px-3 py-2 bg-green-600 hover:bg-green-700 disabled:bg-zinc-700
                     text-white text-sm rounded transition-colors disabled:cursor-not-allowed"
        >
          Query
        </button>
      </div>

      {/* SQL Editor (exec) */}
      <div className="border border-zinc-700 rounded-lg overflow-hidden">
        <div className="flex items-center justify-between px-3 py-2 bg-zinc-800 border-b border-zinc-700">
          <span className="text-sm text-zinc-400">SQL Editor (CREATE / INSERT / UPDATE / DELETE)</span>
          <button
            onClick={handleExecute}
            disabled={loading || !sql.trim()}
            className="px-3 py-1 bg-orange-600 hover:bg-orange-700 disabled:bg-zinc-700
                       text-white text-sm rounded transition-colors disabled:cursor-not-allowed"
          >
            {loading ? 'Running...' : 'Exec (Ctrl+Enter)'}
          </button>
        </div>
        <textarea
          value={sql}
          onChange={e => handleChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"
          className="w-full h-32 p-3 bg-zinc-900 text-zinc-100 font-mono text-sm
                     resize-y focus:outline-none placeholder-zinc-600"
          spellCheck={false}
        />
      </div>
    </div>
  )
}
