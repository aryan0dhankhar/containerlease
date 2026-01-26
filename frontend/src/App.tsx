import { useState, useCallback } from 'react'
import './App.css'
import { ProvisionForm } from './components/ProvisionForm'
import { ContainerList } from './components/ContainerList'

const BUILD_TIME = new Date().toLocaleString()

function App() {
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [containerCount, setContainerCount] = useState(0)

  const handleContainerProvisioned = useCallback(() => {
    setRefreshTrigger((prev) => prev + 1)
  }, [])

  return (
    <div className="app">
      {/* Top Header */}
      <header className="app-header">
        <div className="header-left">
          <span>Container</span>
          <span style={{ color: '#38bdf8' }}>Lease</span>
        </div>
        <div className="header-right">
          <div className="stat">
            <span className="stat-label">Active</span>
            <span className="stat-value">{containerCount}</span>
          </div>
          <div className="stat">
            <span className="stat-label">Uptime</span>
            <span className="stat-value">∞</span>
          </div>
          <div className="stat">
            <span className="stat-label">Build</span>
            <span className="stat-value" style={{ fontSize: '0.85rem' }}>{BUILD_TIME.split(',')[0]}</span>
          </div>
        </div>
      </header>

      {/* Control Bar / Launchpad */}
      <div className="control-bar">
        <ProvisionForm onProvisioned={handleContainerProvisioned} />
      </div>

      {/* Main Content */}
      <main className="app-main">
        <div className="container-grid">
          <ContainerList key={refreshTrigger} onCountChange={setContainerCount} />
        </div>
      </main>
    </div>
  )
}

export default App
