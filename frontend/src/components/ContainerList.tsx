import React, { useEffect, useState } from 'react'
import { containerApi } from '../services/containerApi'
import type { Container } from '../types/container'
import { ContainerLogs } from './ContainerLogs'

interface ContainerListProps {
  onCountChange?: (count: number) => void
}

export const ContainerList: React.FC<ContainerListProps> = ({ onCountChange }) => {
  const [containers, setContainers] = useState<Container[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timers, setTimers] = useState<Record<string, number>>({})
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(null)

  useEffect(() => {
    fetchContainers()
    const interval = setInterval(fetchContainers, 5000) // Refresh every 5s
    return () => clearInterval(interval)
  }, [])

  // Timer countdown effect
  useEffect(() => {
    const timerInterval = setInterval(() => {
      setTimers((prev) => {
        const updated = { ...prev }
        Object.keys(updated).forEach((id) => {
          updated[id] = Math.max(0, updated[id] - 1)
        })
        return updated
      })
    }, 1000)
    return () => clearInterval(timerInterval)
  }, [])

  const fetchContainers = async () => {
    try {
      const containerList = await containerApi.getContainers()
      setContainers(containerList)
      onCountChange?.(containerList.length)

      // Initialize timers for each container
      const newTimers: Record<string, number> = {}
      containerList.forEach((c: Container) => {
        const expiryTime = new Date(c.expiryAt).getTime()
        const now = Date.now()
        const secondsLeft = Math.max(0, Math.floor((expiryTime - now) / 1000))
        newTimers[c.id] = secondsLeft
      })
      setTimers(newTimers)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch containers')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (window.confirm('Delete this container?')) {
      try {
        await containerApi.deleteContainer(id)
        const event = new CustomEvent('toast', {
          detail: {
            type: 'success',
            message: 'Container deleted successfully'
          }
        })
        window.dispatchEvent(event)
        await fetchContainers()
      } catch (err) {
        const event = new CustomEvent('toast', {
          detail: {
            type: 'error',
            message: err instanceof Error ? err.message : 'Failed to delete container'
          }
        })
        window.dispatchEvent(event)
      }
    }
  }

  const formatTime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  }

  const getTimerClass = (seconds: number): string => {
    if (seconds < 300) return 'critical' // < 5 min
    if (seconds < 900) return 'warning' // < 15 min
    return ''
  }

  if (loading && containers.length === 0) {
    return (
      <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '2rem', color: '#6e7681' }}>
        Loading containers...
      </div>
    )
  }

  if (error && containers.length === 0) {
    return (
      <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '2rem', color: '#f87171' }}>
        Error: {error}
      </div>
    )
  }

  if (containers.length === 0) {
    return (
      <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '3rem', color: '#6e7681' }}>
        <p style={{ fontSize: '1.1rem', marginBottom: '0.5rem' }}>No active containers</p>
        <p style={{ fontSize: '0.9rem' }}>Provision one using the control bar above to get started</p>
      </div>
    )
  }

  return (
    <>
      {containers.map((container) => (
        <div key={container.id} className={`container-card ${getTimerClass(timers[container.id] || 0)}`}>
          <div className="card-header">
            <div>
              <div className="card-title">{container.imageType}</div>
              <div className="card-id">{container.id.substring(0, 12)}</div>
            </div>
            <span className={`status-badge ${container.status}`}>
              {container.status === 'running' && '●'}
              {container.status === 'terminated' && '✓'}
              {container.status}
            </span>
          </div>

          <div className="card-timer" style={{ color: 
            container.status === 'terminated' ? '#6e7681' :
            getTimerClass(timers[container.id] || 0) === 'critical' ? '#f87171' :
            getTimerClass(timers[container.id] || 0) === 'warning' ? '#facc15' :
            '#38bdf8'
          }}>
            {formatTime(timers[container.id] || 0)}
          </div>

          <div className="card-meta">
            <div className="meta-item">
              <span className="meta-label">CPU</span>
              <span className="meta-value">{container.cpuMilli}m</span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Memory</span>
              <span className="meta-value">{container.memoryMB}MB</span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Cost</span>
              <span className="meta-value">${container.cost?.toFixed(2) || '0.00'}</span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Created</span>
              <span className="meta-value">{new Date(container.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
            </div>
          </div>

          <div className="card-actions">
            <button
              onClick={() => setSelectedContainerId(container.id)}
              className="btn btn-logs"
              disabled={container.status !== 'running'}
            >
              Logs
            </button>
            <button
              onClick={() => handleDelete(container.id)}
              className="btn btn-terminate"
            >
              Terminate
            </button>
          </div>
        </div>
      ))}
      {selectedContainerId && (
        <ContainerLogs
          containerId={selectedContainerId}
          onClose={() => setSelectedContainerId(null)}
        />
      )}
    </>
  )
}

export default ContainerList

