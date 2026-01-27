import { useState, useCallback, useEffect } from 'react'
import './App.css'
import { ProvisionForm } from './components/ProvisionForm'
import { ContainerList } from './components/ContainerList'
import { LoginForm } from './components/LoginForm'
import type { LoginResponse } from './types/container'

const BUILD_TIME = new Date().toLocaleString()

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [user, setUser] = useState<LoginResponse | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [containerCount, setContainerCount] = useState(0)

  // Check for existing token on mount
  useEffect(() => {
    const token = localStorage.getItem('token')
    const userStr = localStorage.getItem('user')
    if (token && userStr) {
      try {
        const userData = JSON.parse(userStr)
        setIsAuthenticated(true)
        setUser({ ...userData, token, expiresAt: '' })
      } catch (e) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
      }
    }
  }, [])

  const handleLoginSuccess = useCallback((token: string, userData: LoginResponse) => {
    setIsAuthenticated(true)
    setUser(userData)
  }, [])

  const handleLogout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setIsAuthenticated(false)
    setUser(null)
  }, [])

  const handleContainerProvisioned = useCallback(() => {
    setRefreshTrigger((prev) => prev + 1)
  }, [])

  if (!isAuthenticated) {
    return <LoginForm onLoginSuccess={handleLoginSuccess} />
  }

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
            <span className="stat-label">User</span>
            <span className="stat-value" style={{ fontSize: '0.85rem' }}>{user?.userId}</span>
          </div>
          <div className="stat">
            <span className="stat-label">Build</span>
            <span className="stat-value" style={{ fontSize: '0.85rem' }}>{BUILD_TIME.split(',')[0]}</span>
          </div>
          <button 
            onClick={handleLogout}
            style={{
              padding: '6px 12px',
              background: '#ef4444',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: 500,
            }}
          >
            Logout
          </button>
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
