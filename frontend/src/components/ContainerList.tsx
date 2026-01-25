import React, { useEffect, useState } from 'react'
import { containerApi } from '../services/containerApi'
import type { Container } from '../types/container'
import './ContainerList.css'
import { ContainerLogs } from './ContainerLogs'

export const ContainerList: React.FC = () => {
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
        await fetchContainers()
      } catch (err) {
        alert(
          err instanceof Error ? err.message : 'Failed to delete container'
        )
      }
    }
  }

  const formatTime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) return `${hours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m ${secs}s`
    return `${secs}s`
  }

  if (loading && containers.length === 0) {
    return <div className="loading">Loading containers...</div>
  }

  if (error && containers.length === 0) {
    return <div className="error-message">Error: {error}</div>
  }

  return (
    <div className="container-list">
      <h2>Active Containers ({containers.length})</h2>
      {containers.length === 0 ? (
        <p className="empty-state">No active containers. Provision one to get started!</p>
      ) : (
        <div className="table-wrapper">
          <table className="containers-table">
            <thead>
              <tr>
                <th>Container ID</th>
                <th>Image</th>
                <th>Status</th>
                  <th>CPU</th>
                  <th>Memory</th>
                <th>Cost</th>
                <th>Created</th>
                <th>Time Remaining</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {containers.map((container) => (
                <tr key={container.id} className="container-row">
                  <td className="container-id">
                    <code>{container.id.substring(0, 12)}</code>
                  </td>
                  <td className="image-type">
                    <span className="image-badge">{container.imageType}</span>
                  </td>
                  <td className="status">
                    <span className={`status-badge status-${container.status}`}>
                      {container.status}
                    </span>
                  </td>
                    <td className="cpu">
                      {container.cpuMilli ? `${container.cpuMilli}m` : '-'}
                    </td>
                    <td className="memory">
                      {container.memoryMB ? `${container.memoryMB} MB` : '-'}
                    </td>
                  <td className="cost">
                      <span className={`cost-badge ${container.status === 'terminated' ? 'final-cost' : ''}`}>
                        ${container.cost?.toFixed(2) || '0.00'}
                        {container.status === 'terminated' && <small> (final)</small>}
                      </span>
                  </td>
                  <td className="created-at">
                    {new Date(container.createdAt).toLocaleTimeString()}
                  </td>
                  <td className="time-remaining">
                    <span
                      className={`timer ${timers[container.id] < 300 ? 'warning' : ''}`}
                    >
                      {formatTime(timers[container.id] || 0)}
                    </span>
                  </td>
                  <td className="actions">
                      <button
                        onClick={() => setSelectedContainerId(container.id)}
                        className="btn btn-secondary btn-small"
                        disabled={container.status !== 'running'}
                        style={{ marginRight: '8px' }}
                      >
                        Logs
                      </button>
                    <button
                      onClick={() => handleDelete(container.id)}
                      className="btn btn-danger btn-small"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {selectedContainerId && (
        <ContainerLogs
          containerId={selectedContainerId}
          onClose={() => setSelectedContainerId(null)}
        />
      )}
    </div>
  )
}

export default ContainerList

