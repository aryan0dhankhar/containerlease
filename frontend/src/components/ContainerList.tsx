import React, { useEffect, useState } from 'react'
import { containerApi } from '../services/containerApi'
import type { Container } from '../types/container'
import './ContainerList.css'

export const ContainerList: React.FC = () => {
  const [containers, setContainers] = useState<Container[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timers, setTimers] = useState<Record<string, number>>({})

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
      const response = await fetch('http://localhost:8080/api/containers')
      if (!response.ok) throw new Error('Failed to fetch containers')
      
      const data = await response.json()
      const containerList = data.containers || []
      setContainers(containerList)

      // Initialize timers for each container
      const newTimers: Record<string, number> = {}
      containerList.forEach((c: Container) => {
        newTimers[c.id] = Math.max(0, c.expiresIn || 0)
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
        await fetch(`http://localhost:8080/api/containers/${id}`, {
          method: 'DELETE',
        })
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
    </div>
  )
}

export default ContainerList

