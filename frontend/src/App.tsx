import { useState, useCallback, useEffect } from 'react'
import './styles/design-system.css'
import './styles/dashboard.css'
import './styles/provision-form.css'
import './styles/app-enhanced.css'
import { CommandCenter } from './components/CommandCenter'
import { ProvisionFormEnhanced } from './components/ProvisionFormEnhanced'
import { HomePage } from './components/HomePage'
import type { LoginResponse } from './types/container'

const BUILD_TIME = new Date().toLocaleString()

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [user, setUser] = useState<LoginResponse | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [showProvisionForm, setShowProvisionForm] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Check for existing token on mount
  useEffect(() => {
    try {
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
    } catch (err) {
      console.error('Error in App mount:', err)
      setError('Failed to initialize app')
    }
  }, [])

  const handleLoginSuccess = useCallback((token: string, userData: LoginResponse) => {
    try {
      setIsAuthenticated(true)
      setUser(userData)
      setError(null)
    } catch (err) {
      console.error('Error in handleLoginSuccess:', err)
      setError('Login succeeded but failed to update UI')
    }
  }, [])

  const handleLogout = useCallback(() => {
    try {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      setIsAuthenticated(false)
      setUser(null)
      setError(null)
    } catch (err) {
      console.error('Error in handleLogout:', err)
    }
  }, [])

  const handleContainerProvisioned = useCallback(() => {
    try {
      setRefreshTrigger((prev) => prev + 1)
      setShowProvisionForm(false)
    } catch (err) {
      console.error('Error in handleContainerProvisioned:', err)
      setError('Container created but failed to update dashboard')
    }
  }, [])

  if (error) {
    return (
      <div style={{ 
        padding: '2rem', 
        color: 'red', 
        fontSize: '18px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        background: '#0f172a'
      }}>
        <div>Error: {error}</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <HomePage onLoginSuccess={handleLoginSuccess} />
  }

  return (
    <div className="app-enhanced">
      {/* Minimalist Top Bar */}
      <header className="app-header-enhanced">
        <div className="header-brand">
          <h1>
            Container<span className="brand-accent">Lease</span>
          </h1>
        </div>

        <nav className="header-nav">
          <button 
            className="btn btn-primary btn-sm"
            onClick={() => setShowProvisionForm(true)}
          >
            + Create Container
          </button>
          <button 
            className="btn btn-secondary btn-sm"
            onClick={handleLogout}
          >
            {user?.userId ? user.userId.substring(0, 8) : 'user'} • Logout
          </button>
        </nav>
      </header>

      {/* Main Content */}
      <main className="app-main-enhanced">
        {showProvisionForm ? (
          <ProvisionFormEnhanced
            onSuccess={handleContainerProvisioned}
            onCancel={() => setShowProvisionForm(false)}
          />
        ) : (
          <CommandCenter
            refreshTrigger={refreshTrigger}
          />
        )}
      </main>

      {/* Footer with Build Info */}
      <footer className="app-footer">
        <small>Built {BUILD_TIME} • All containers are ephemeral and automatically destroyed at expiration</small>
      </footer>
    </div>
  )
}

export default App
