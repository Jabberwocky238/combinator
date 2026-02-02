import { CombinatorProvider, useCombinator } from './context/CombinatorContext'
import { AppBar } from './components/AppBar'
import { ConnectPanel } from './components/ConnectPanel'
import { ServiceListPanel } from './components/ServiceListPanel'

function AppContent() {
  const { endpoint, isConnected } = useCombinator()

  return (
    <div className="min-h-screen bg-zinc-900">
      <AppBar />
      <main className="pt-14">
        {!endpoint ? (
          <ConnectPanel />
        ) : (
          <ServiceListPanel />
        )}
      </main>
    </div>
  )
}

function App() {
  return (
    <CombinatorProvider>
      <AppContent />
    </CombinatorProvider>
  )
}

export default App
