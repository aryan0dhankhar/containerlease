import React, { useState, useMemo } from 'react'
import { containerApi } from '../services/containerApi'
import './ProvisionForm.css'

interface ProvisionFormProps {
  onProvisioned?: () => void
}

const calculateCost = (imageType: string, durationMinutes: number): number => {
  const hourlyRate = imageType === 'ubuntu' ? 0.04 : 0.01
  return (hourlyRate * durationMinutes) / 60
}

export const ProvisionForm: React.FC<ProvisionFormProps> = ({
  onProvisioned,
}) => {
  const [imageType, setImageType] = useState<string>('ubuntu')
  const [duration, setDuration] = useState<number>(30)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [cpuMilli, setCpuMilli] = useState<number>(500)
  const [memoryMB, setMemoryMB] = useState<number>(512)
  const [logDemo, setLogDemo] = useState<boolean>(true)

  const minDuration = 5
  const maxDuration = 120

  const cost = useMemo(
    () => calculateCost(imageType, duration),
    [imageType, duration]
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    setSuccess(null)

    try {
      const result = await containerApi.provision(imageType, duration, cpuMilli, memoryMB, logDemo)
      setSuccess(
        `Container ${result.id.substring(0, 12)} provisioned successfully!`
      )
      setImageType('ubuntu')
      setDuration(30)
      setCpuMilli(500)
      setMemoryMB(512)
      setLogDemo(true)
      onProvisioned?.()
      setTimeout(() => setSuccess(null), 5000)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="provision-form">
      <div className="form-group">
        <label htmlFor="imageType">Image Type:</label>
        <select
          id="imageType"
          value={imageType}
          onChange={(e) => setImageType(e.target.value)}
          disabled={loading}
          className="form-input"
        >
          <option value="ubuntu">Ubuntu 22.04</option>
          <option value="alpine">Alpine Linux</option>
        </select>
      </div>

      <div className="form-group">
        <label htmlFor="duration">Duration (minutes):</label>
        <input
          id="duration"
          type="number"
          min={minDuration}
          max={maxDuration}
          value={duration}
          onChange={(e) => setDuration(parseInt(e.target.value))}
          disabled={loading}
          className="form-input"
        />
        <small>Min: {minDuration} min, Max: {maxDuration} min</small>
      </div>

      <div className="form-group">
        <label htmlFor="cpuMilli">CPU (millicores):</label>
        <select
          id="cpuMilli"
          value={cpuMilli}
          onChange={(e) => setCpuMilli(parseInt(e.target.value))}
          disabled={loading}
          className="form-input"
        >
          <option value="250">250m (0.25 CPU)</option>
          <option value="500">500m (0.5 CPU)</option>
          <option value="1000">1000m (1 CPU)</option>
          <option value="2000">2000m (2 CPU)</option>
        </select>
      </div>

      <div className="form-group">
        <label htmlFor="memoryMB">Memory (MB):</label>
        <select
          id="memoryMB"
          value={memoryMB}
          onChange={(e) => setMemoryMB(parseInt(e.target.value))}
          disabled={loading}
          className="form-input"
        >
          <option value="256">256 MB</option>
          <option value="512">512 MB</option>
          <option value="1024">1024 MB (1 GB)</option>
          <option value="2048">2048 MB (2 GB)</option>
        </select>
      </div>

      <div className="form-group">
        <label htmlFor="logDemo" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <input
            id="logDemo"
            type="checkbox"
            checked={logDemo}
            onChange={(e) => setLogDemo(e.target.checked)}
            disabled={loading}
          />
          <span>Enable demo logs (emit log lines for testing)</span>
        </label>
        <small>Recommended for testing the log viewer</small>
      </div>

      <div className="cost-display">
        <strong>Estimated Cost: ${cost.toFixed(2)}</strong>
      </div>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      <button type="submit" disabled={loading} className="btn btn-primary">
        {loading ? 'Provisioning...' : 'Provision Container'}
      </button>
    </form>
  )
}
