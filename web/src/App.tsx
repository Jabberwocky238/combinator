import { Routes, Route, Navigate } from 'react-router-dom'
import { CombinatorProvider, useCombinator } from './context/CombinatorContext'
import { AppBar } from './components/AppBar'
import { ConnectPanel } from './components/ConnectPanel'
import { ServiceListPanel } from './components/ServiceListPanel'
import { RDBDetailPanel } from './components/RDBDetailPanel'

function AppContent() {
  const { endpoint } = useCombinator()

  if (!endpoint) {
    return (
      <div className="min-h-screen bg-zinc-900">
        <AppBar />
        <main className="pt-14">
          <ConnectPanel />
        </main>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-zinc-900">
      <AppBar />
      <main className="pt-14">
        <Routes>
          <Route path="/" element={<ServiceListPanel />} />
          <Route path="/rdb/:id" element={<RDBDetailPanel />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
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
