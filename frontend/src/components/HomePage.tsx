import React, { useState } from 'react'
import { LoginForm } from './LoginForm'
import type { LoginResponse } from '../types/container'
import './HomePage.css'

interface HomePageProps {
  onLoginSuccess: (token: string, user: LoginResponse) => void
}

export const HomePage: React.FC<HomePageProps> = ({ onLoginSuccess }) => {
  const [showAuth, setShowAuth] = useState(false)
  const [authMode, setAuthMode] = useState<'login' | 'signup'>('login')

  if (showAuth) {
    return (
      <LoginForm onLoginSuccess={onLoginSuccess} initialMode={authMode} />
    )
  }

  return (
    <div className="homepage">
      {/* Hero Section */}
      <section className="hero">
        <div className="hero-content">
          <h1>
            <span>Container</span>
            <span className="hero-highlight">Lease</span>
          </h1>
          <p className="tagline">On-Demand Container Provisioning Platform</p>
          <p className="description">
            Provision, manage, and monitor Docker containers with ease.
          </p>

          <div className="cta-buttons">
            <button
              className="btn-primary btn-large"
              onClick={() => {
                setAuthMode('login')
                setShowAuth(true)
              }}
            >
              Sign In
            </button>
            <button
              className="btn-secondary btn-large"
              onClick={() => {
                setAuthMode('signup')
                setShowAuth(true)
              }}
            >
              Get Started
            </button>
          </div>
        </div>

        <div className="hero-graphic">
          <div className="container-visual">
            <div className="container-icon">
              <span>⬡</span>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="features">
        <h2>Why ContainerLease?</h2>
        <div className="features-grid">
          <div className="feature-card">
            <div className="feature-icon">⚡</div>
            <h3>Instant Provisioning</h3>
            <p>Spin up containers in seconds with pre-configured templates</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">📊</div>
            <h3>Real-time Monitoring</h3>
            <p>Track logs, resource usage, and container status live</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">🔒</div>
            <h3>Secure & Isolated</h3>
            <p>Enterprise-grade security with complete container isolation</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">🛠️</div>
            <h3>Easy Management</h3>
            <p>Intuitive dashboard to manage all your containers</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">🚀</div>
            <h3>Scale Effortlessly</h3>
            <p>Run multiple containers and scale up as your needs grow</p>
          </div>
        </div>
      </section>



      {/* Footer */}
      <footer className="homepage-footer">
        <p>&copy; 2026 ContainerLease. All rights reserved.</p>
      </footer>
    </div>
  )
}
