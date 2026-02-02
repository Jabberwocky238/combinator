import { useCombinator } from '../context/CombinatorContext'
import { Delete28Filled } from '@ricons/fluent'

export function AppBar() {
  const { endpoint, isConnected, connectionError, clearEndpoint } = useCombinator()

  return (
    <header className="fixed top-0 left-0 right-0 h-14 bg-zinc-900 border-b border-zinc-700 flex items-center justify-between px-4 z-50">
      <div className="flex items-center gap-3">
        <span className="font-semibold text-white">Combinator</span>
        <div className="h-4 w-px bg-zinc-600" />
        {endpoint ? (
          <div className="flex items-center gap-2">
            <span className="text-zinc-400 text-sm">{endpoint}</span>
            <StatusIndicator connected={isConnected} error={connectionError} />
          </div>
        ) : (
          <span className="text-zinc-500 text-sm">No endpoint configured</span>
        )}
      </div>

      <div className="flex items-center gap-2">
        {endpoint && (
          <Delete28Filled onClick={clearEndpoint} className="w-7 h-7 text-zinc-400 cursor-pointer hover:text-white" />
        )}
      </div>
    </header>
  )
}


function StatusIndicator({ connected, error }: { connected: boolean; error: string | null }) {
  if (connected) {
    return <span className="w-2 h-2 rounded-full bg-green-500" title="Connected" />
  }
  if (error) {
    return <span className="w-2 h-2 rounded-full bg-red-500" title={error} />
  }
  return <span className="w-2 h-2 rounded-full bg-yellow-500 animate-pulse" title="Connecting..." />
}


