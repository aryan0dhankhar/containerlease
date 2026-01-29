import React, { useState, useEffect } from 'react'
import { containerApi } from '../services/containerApi'
import type { Container } from '../types/container'
import '../styles/dashboard.css'

interface CommandCenterProps {
  refreshTrigger: number
  onContainerSelect?: (container: Container) => void
}

export const CommandCenter: React.FC<CommandCenterProps> = ({ refreshTrigger, onContainerSelect }) => {
  const [containers, setContainers] = useState<Container[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedContainer, setSelectedContainer] = useState<string | null>(null)
  const [sortBy, setSortBy] = useState<'expiry' | 'created' | 'status'>('expiry')
  const [filterStatus, setFilterStatus] = useState<'all' | 'running' | 'pending' | 'failed'>('all')
  const [logsContainerId, setLogsContainerId] = useState<string | null>(null)
  const [logs, setLogs] = useState<string>('')
  const [logsLoading, setLogsLoading] = useState(false)

  useEffect(() => {
    const fetchContainers = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await containerApi.getContainers()
        setContainers(data || [])
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to load containers'
        console.error('Error fetching containers:', err)
        setError(errorMsg)
      } finally {
        setLoading(false)
      }
    }

    fetchContainers()
    const interval = setInterval(fetchContainers, 1000) // Refresh every 1 second
    return () => clearInterval(interval)
  }, [refreshTrigger])

  const getTimeRemaining = (expiryTime: string): { time: string; percentage: number; status: 'critical' | 'warning' | 'normal' } => {
    const now = new Date().getTime()
    const expiry = new Date(expiryTime).getTime()
    const diff = expiry - now

    if (diff <= 0) return { time: 'EXPIRED', percentage: 0, status: 'critical' }
    
    const minutes = Math.floor(diff / 60000)
    const seconds = Math.floor((diff % 60000) / 1000)
    
    let status: 'critical' | 'warning' | 'normal' = 'normal'
    if (minutes <= 5) status = 'critical'
    else if (minutes <= 15) status = 'warning'
    
    return {
      time: `${minutes}:${seconds.toString().padStart(2, '0')}`,
      percentage: (diff / (120 * 60000)) * 100, // Assume 120 min max
      status
    }
  }

  const getStatusBadge = (status: string) => {
    const badges: { [key: string]: { label: string; class: string; icon: string } } = {
      running: { label: 'RUNNING', class: 'badge-success', icon: '◉' },
      pending: { label: 'PENDING', class: 'badge-warning', icon: '◐' },
      failed: { label: 'FAILED', class: 'badge-critical', icon: '✕' },
      terminated: { label: 'TERMINATED', class: 'badge-info', icon: '■' },
    }
    const badge = badges[status] || badges['pending']
    return badge
  }

  const handleDestroy = async (containerId: string) => {
    if (!confirm('Are you sure you want to destroy this container? This action cannot be undone.')) {
      return
    }
    try {
      const token = localStorage.getItem('token')
      const response = await fetch(`http://localhost:8080/api/containers/${containerId}`, {
        method: 'DELETE',
        headers: token ? { 'Authorization': `Bearer ${token}` } : {},
      })
      
      if (response.ok) {
        // Remove container from local state
        setContainers(prev => prev.filter(c => c.id !== containerId))
      } else {
        alert('Failed to destroy container')
      }
    } catch (error) {
      console.error('Error destroying container:', error)
      alert('Failed to destroy container')
    }
  }

  const handleViewLogs = async (containerId: string) => {
    setLogsContainerId(containerId)
    setLogsLoading(true)
    setLogs('')
    try {
      const token = localStorage.getItem('token')
      const user = localStorage.getItem('user')
      
      if (!token || !user) {
        setLogs('Error: Not authenticated. Please log in.')
        setLogsLoading(false)
        return
      }
      
      const response = await fetch(`http://localhost:8080/api/logs?container=${containerId}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      })
      
      if (response.ok) {
        const data = await response.json()
        setLogs(data.logs || 'No logs available')
      } else if (response.status === 401) {
        setLogs('Error: Not authenticated. Please log in again.')
      } else if (response.status === 404) {
        setLogs('Error: Container not found. It may have expired or been deleted.')
      } else {
        const errorData = await response.json().catch(() => ({}))
        setLogs(`Error (${response.status}): ${errorData.error || 'Failed to fetch logs'}`)
      }
    } catch (err) {
      console.error('Error fetching logs:', err)
      setLogs(`Error: ${err instanceof Error ? err.message : 'Failed to fetch logs'}`)
    } finally {
      setLogsLoading(false)
    }
  }

  const filteredContainers = containers.filter(c => {
    if (filterStatus === 'all') return true
    return c.status === filterStatus
  })

  const sortedContainers = [...filteredContainers].sort((a, b) => {
    if (sortBy === 'expiry') {
      return new Date(a.expiryAt).getTime() - new Date(b.expiryAt).getTime()
    } else if (sortBy === 'created') {
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    }
    return 0
  })

  return (
    <div className="command-center">
      {/* Header with Controls */}
      <div className="cc-header">
        <div className="cc-title-section">
          <h1>Container Command Center</h1>
          <p className="cc-subtitle">
            {filteredContainers.length} active · {containers.filter(c => c.status === 'running').length} running
          </p>
        </div>

        <div className="cc-controls">
          <div className="cc-filter-group">
            <label htmlFor="sort">Sort By:</label>
            <select
              id="sort"
              className="form-select form-select-sm"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as typeof sortBy)}
            >
              <option value="expiry">Expiring Soonest</option>
              <option value="created">Most Recent</option>
              <option value="status">Status</option>
            </select>
          </div>

          <div className="cc-filter-group">
            <label htmlFor="filter">Filter:</label>
            <select
              id="filter"
              className="form-select form-select-sm"
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value as typeof filterStatus)}
            >
              <option value="all">All Containers</option>
              <option value="running">Running</option>
              <option value="pending">Pending</option>
              <option value="failed">Failed</option>
            </select>
          </div>
        </div>
      </div>

      {/* Status Alert */}
      {error && (
        <div className="alert alert-critical">
          <strong>⚠ Error:</strong> {error}
        </div>
      )}

      {loading && sortedContainers.length === 0 ? (
        <div className="cc-loading">
          <div className="spinner"></div>
          <p>Loading containers...</p>
        </div>
      ) : sortedContainers.length === 0 ? (
        <div className="cc-empty">
          <div className="cc-empty-icon">∅</div>
          <h3>No Containers</h3>
          <p>Create a new container to get started</p>
        </div>
      ) : (
        <div className="cc-grid">
          {sortedContainers.map((container) => {
            const { time, percentage, status: ttlStatus } = getTimeRemaining(container.expiryAt)
            const statusBadge = getStatusBadge(container.status)
            const isSelected = selectedContainer === container.id

            return (
              <div
                key={container.id}
                className={`cc-card ${isSelected ? 'cc-card-selected' : ''} cc-card-${container.status}`}
                onClick={() => {
                  setSelectedContainer(container.id)
                  onContainerSelect?.(container)
                }}
                role="button"
                tabIndex={0}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    setSelectedContainer(container.id)
                    onContainerSelect?.(container)
                  }
                }}
              >
                {/* Card Header */}
                <div className="cc-card-header">
                  <div className="cc-card-title">
                    <span className="status-indicator" style={{
                      backgroundColor: container.status === 'running' ? 'var(--color-success)' :
                                       container.status === 'pending' ? 'var(--color-warning)' :
                                       'var(--color-critical)'
                    }}></span>
                    <div>
                      <h4>{container.id.substring(0, 12)}</h4>
                      <span className="cc-image-tag">{container.imageType}</span>
                    </div>
                  </div>
                  <div className={`badge ${statusBadge.class}`}>
                    {statusBadge.icon} {statusBadge.label}
                  </div>
                </div>

                {/* TTL Progress Bar */}
                <div className="cc-ttl-section">
                  <div className="cc-ttl-header">
                    <span className="cc-ttl-label">Time to Expiry</span>
                    <span className={`cc-ttl-time cc-ttl-${ttlStatus}`}>{time}</span>
                  </div>
                  <div className={`cc-progress-bar cc-progress-${ttlStatus}`}>
                    <div
                      className="cc-progress-fill"
                      style={{ width: `${Math.max(percentage, 5)}%` }}
                    ></div>
                  </div>
                </div>

                {/* Resources & Info Grid */}
                <div className="cc-info-grid">
                  <div className="cc-info-item">
                    <span className="cc-info-label">CPU</span>
                    <span className="cc-info-value">{container.cpuMilli || 500}m</span>
                  </div>
                  <div className="cc-info-item">
                    <span className="cc-info-label">Memory</span>
                    <span className="cc-info-value">{container.memoryMB || 512}MB</span>
                  </div>
                  <div className="cc-info-item">
                    <span className="cc-info-label">Status</span>
                    <span className="cc-info-value" style={{
                      color: container.status === 'running' ? 'var(--color-success)' : 'var(--color-warning)'
                    }}>
                      {container.status.toUpperCase()}
                    </span>
                  </div>
                  <div className="cc-info-item">
                    <span className="cc-info-label">Created</span>
                    <span className="cc-info-value">{new Date(container.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                  </div>
                </div>

                {/* Isolation Indicators */}
                <div className="cc-isolation">
                  <div className="cc-isolation-badge">
                    <span className="isolation-icon">🔒</span> Isolated
                  </div>
                  <div className="cc-isolation-badge">
                    <span className="isolation-icon">🌐</span> Ephemeral
                  </div>
                  <div className="cc-isolation-badge">
                    <span className="isolation-icon">✓</span> Sandboxed
                  </div>
                </div>

                {/* Quick Actions */}
                {container.status === 'running' && (
                  <div className="cc-card-actions">
                    <button 
                      className="btn btn-sm btn-secondary"
                      onClick={() => handleViewLogs(container.id)}
                    >
                      Logs
                    </button>
                    <button 
                      className="btn btn-sm btn-danger"
                      onClick={() => handleDestroy(container.id)}
                    >
                      Destroy
                    </button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Logs Modal */}
      {logsContainerId && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.7)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1000,
        }}>
          <div style={{
            backgroundColor: 'var(--color-bg-secondary)',
            borderRadius: '8px',
            padding: '2rem',
            maxWidth: '800px',
            width: '90%',
            maxHeight: '80vh',
            display: 'flex',
            flexDirection: 'column',
            border: '1px solid var(--color-border)',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '1rem' }}>
              <h3>Container Logs</h3>
              <button 
                onClick={() => setLogsContainerId(null)}
                style={{
                  background: 'none',
                  border: 'none',
                  color: 'var(--color-text-primary)',
                  fontSize: '24px',
                  cursor: 'pointer',
                }}
              >
                ✕
              </button>
            </div>
            <div style={{
              flex: 1,
              overflow: 'auto',
              backgroundColor: 'var(--color-bg-primary)',
              padding: '1rem',
              borderRadius: '4px',
              fontFamily: 'monospace',
              fontSize: '12px',
              color: 'var(--color-text-secondary)',
              lineHeight: '1.5',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}>
              {logsLoading ? '⟳ Loading logs...' : logs}
            </div>
            <button 
              className="btn btn-secondary"
              onClick={() => setLogsContainerId(null)}
              style={{ marginTop: '1rem' }}
            >
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
