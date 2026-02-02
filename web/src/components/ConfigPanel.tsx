import { useState } from 'react'
import { useCombinator } from '../context/CombinatorContext'

export function ConfigPanel() {
  const { endpoint, setEndpoint, clearEndpoint, isConnected, connectionError } = useCombinator()
  const [inputValue, setInputValue] = useState(endpoint || '')

  const handleConnect = () => {
    if (inputValue.trim()) {
      setEndpoint(inputValue.trim())
    }
  }

  return (
    <div className="config-panel">
      <h3>API Endpoint</h3>
      <div className="config-input-row">
        <input
          type="text"
          value={inputValue}
          onChange={e => setInputValue(e.target.value)}
          placeholder="http://localhost:8899"
          onKeyDown={e => e.key === 'Enter' && handleConnect()}
        />
        <button onClick={handleConnect}>Connect</button>
        {endpoint && <button onClick={clearEndpoint}>Clear</button>}
      </div>
      <div className="connection-status">
        {!endpoint && <span className="status-waiting">Waiting for endpoint...</span>}
        {endpoint && isConnected && <span className="status-connected">Connected</span>}
        {endpoint && !isConnected && connectionError && (
          <span className="status-error">Error: {connectionError}</span>
        )}
        {endpoint && !isConnected && !connectionError && (
          <span className="status-connecting">Connecting...</span>
        )}
      </div>
    </div>
  )
}
