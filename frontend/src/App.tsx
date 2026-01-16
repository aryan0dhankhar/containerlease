import { useState, useCallback } from 'react'
import './App.css'
import { ProvisionForm } from './components/ProvisionForm'
import { ContainerList } from './components/ContainerList'

function App() {
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const handleContainerProvisioned = useCallback(() => {
    // Trigger container list refresh
    setRefreshTrigger((prev) => prev + 1)
  }, [])

  return (
    <div className="app">
      <header className="app-header">
        <h1>🐳 ContainerLease</h1>
        <p className="subtitle">Provision temporary Docker containers</p>
      </header>

      <main className="app-main">
        <section className="provision-section">
          <h2>Provision Container</h2>
          <ProvisionForm onProvisioned={handleContainerProvisioned} />
        </section>

        <section className="containers-section">
          <ContainerList key={refreshTrigger} />
        </section>
      </main>

      <footer className="app-footer">
        <p>Containers auto-delete after lease expires</p>
      </footer>
    </div>
  )
}

export default App
