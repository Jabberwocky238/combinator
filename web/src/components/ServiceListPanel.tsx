import { useCombinator } from '../context/CombinatorContext'
import { useServiceList, type ServiceInfo } from '../hooks/useServiceList'

export function ServiceListPanel() {
  const { isConnected } = useCombinator()
  const { services, loading, error, refresh } = useServiceList()

  if (!isConnected) return null

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-white">Services</h2>
        <button
          onClick={refresh}
          disabled={loading}
          className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg
                     text-sm transition-colors disabled:opacity-50"
        >
          {loading ? 'Loading...' : 'Refresh'}
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-900/30 border border-red-700 rounded-lg text-red-400">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <ServiceSection title="RDB Instances" items={services.rdb} icon={<DatabaseIcon />} />
        <ServiceSection title="KV Instances" items={services.kv} icon={<KeyIcon />} />
      </div>
    </div>
  )
}

function ServiceSection({ title, items, icon }: {
  title: string
  items: ServiceInfo[]
  icon: React.ReactNode
}) {
  return (
    <div className="bg-zinc-800/50 border border-zinc-700 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-4">
        <span className="text-zinc-400">{icon}</span>
        <h3 className="font-medium text-white">{title}</h3>
        <span className="ml-auto text-zinc-500 text-sm">{items.length}</span>
      </div>

      {items.length === 0 ? (
        <p className="text-zinc-500 text-sm">No instances configured</p>
      ) : (
        <ul className="space-y-2">
          {items.map(item => (
            <li key={item.id} className="flex items-center justify-between p-2 bg-zinc-900/50 rounded">
              <span className="text-white font-mono text-sm">{item.id}</span>
              <span className="text-zinc-500 text-xs uppercase">{item.type}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function DatabaseIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <ellipse cx="12" cy="5" rx="9" ry="3" />
      <path d="M3 5V19A9 3 0 0 0 21 19V5" />
      <path d="M3 12A9 3 0 0 0 21 12" />
    </svg>
  )
}

function KeyIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1" />
      <path d="M7 17v-1a1 1 0 0 1 1-1h1" />
      <circle cx="16" cy="8" r="5" />
    </svg>
  )
}