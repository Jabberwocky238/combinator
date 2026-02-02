import { useState } from 'react'
import { useCombinator } from '../context/CombinatorContext'

export function ConnectPanel() {
  const { setEndpoint } = useCombinator()
  const [inputValue, setInputValue] = useState('http://localhost:8899')

  const handleConnect = () => {
    if (inputValue.trim()) {
      setEndpoint(inputValue.trim())
    }
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold text-white mb-2">
          Connect to Combinator
        </h2>
        <p className="text-zinc-400">
          Enter the API endpoint to get started
        </p>
      </div>

      <div className="flex gap-2 w-full max-w-md">
        <input
          type="text"
          value={inputValue}
          onChange={e => setInputValue(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleConnect()}
          placeholder="http://localhost:8899"
          className="flex-1 px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg
                     text-white placeholder-zinc-500 focus:outline-none focus:border-blue-500"
        />
        <button
          onClick={handleConnect}
          className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg
                     font-medium transition-colors"
        >
          Connect
        </button>
      </div>
    </div>
  )
}
