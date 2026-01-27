import React, { useState } from 'react'
import { containerApi } from '../services/containerApi'
import type { LoginResponse } from '../types/container'
import './LoginForm.css'

interface LoginFormProps {
  onLoginSuccess: (token: string, user: LoginResponse) => void
}

export const LoginForm: React.FC<LoginFormProps> = ({ onLoginSuccess }) => {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const response = await containerApi.login(email, password)
      localStorage.setItem('token', response.token)
      localStorage.setItem('user', JSON.stringify({
        userId: response.userId,
        tenantId: response.tenantId,
      }))
      onLoginSuccess(response.token, response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  // Demo credentials hint
  const demoCredentials = [
    { email: 'demo@example.com', password: 'demo123' },
    { email: 'admin@example.com', password: 'admin123' },
    { email: 'test@example.com', password: 'test123' },
  ]

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>ContainerLease</h1>
        <p className="subtitle">Container Provisioning Platform</p>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="your@email.com"
              required
              disabled={loading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              disabled={loading}
            />
          </div>

          {error && <div className="error-message">{error}</div>}

          <button
            type="submit"
            disabled={loading || !email || !password}
            className="btn-primary"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="demo-section">
          <h3>Demo Credentials</h3>
          <p>Try any of these accounts:</p>
          <ul className="demo-list">
            {demoCredentials.map((cred) => (
              <li key={cred.email}>
                <code>{cred.email}</code> / <code>{cred.password}</code>
                <button
                  type="button"
                  className="btn-demo"
                  onClick={() => {
                    setEmail(cred.email)
                    setPassword(cred.password)
                  }}
                >
                  Use
                </button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
