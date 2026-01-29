import React, { useState } from 'react'
import { containerApi } from '../services/containerApi'
import type { LoginResponse } from '../types/container'
import './LoginForm.css'

interface LoginFormProps {
  onLoginSuccess: (token: string, user: LoginResponse) => void
  initialMode?: 'login' | 'signup'
}

export const LoginForm: React.FC<LoginFormProps> = ({ onLoginSuccess, initialMode = 'login' }) => {
  const [mode, setMode] = useState<'login' | 'signup'>(initialMode)
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (mode === 'login') {
        const response = await containerApi.login(email, password)
        localStorage.setItem('token', response.token)
        localStorage.setItem('user', JSON.stringify({
          userId: response.userId,
          tenantId: response.tenantId,
        }))
        onLoginSuccess(response.token, response)
      } else {
        // Signup mode
        if (password !== confirmPassword) {
          setError('Passwords do not match')
          setLoading(false)
          return
        }

        const response = await containerApi.register(email, username, password)
        localStorage.setItem('token', response.token)
        localStorage.setItem('user', JSON.stringify({
          userId: response.userId,
          tenantId: response.tenantId,
        }))
        onLoginSuccess(response.token, response)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : (mode === 'login' ? 'Login failed' : 'Registration failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>ContainerLease</h1>
        <p className="subtitle">Container Provisioning Platform</p>

        {/* Mode Toggle */}
        <div className="mode-toggle">
          <button
            type="button"
            className={`toggle-btn ${mode === 'login' ? 'active' : ''}`}
            onClick={() => {
              setMode('login')
              setError(null)
              setUsername('')
              setConfirmPassword('')
            }}
          >
            Sign In
          </button>
          <button
            type="button"
            className={`toggle-btn ${mode === 'signup' ? 'active' : ''}`}
            onClick={() => {
              setMode('signup')
              setError(null)
            }}
          >
            Create Account
          </button>
        </div>

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

          {mode === 'signup' && (
            <div className="form-group">
              <label htmlFor="username">Username</label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Choose a username"
                required
                disabled={loading}
                minLength={3}
              />
            </div>
          )}

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
              minLength={mode === 'signup' ? 6 : 1}
            />
          </div>

          {mode === 'signup' && (
            <div className="form-group">
              <label htmlFor="confirmPassword">Confirm Password</label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="••••••••"
                required
                disabled={loading}
                minLength={6}
              />
            </div>
          )}

          {error && <div className="error-message">{error}</div>}

          <button
            type="submit"
            disabled={loading || !email || !password || (mode === 'signup' && (!username || !confirmPassword))}
            className="btn-primary"
          >
            {loading ? (mode === 'login' ? 'Signing in...' : 'Creating account...') : (mode === 'login' ? 'Sign In' : 'Create Account')}
          </button>
        </form>
      </div>
    </div>
  )
}
