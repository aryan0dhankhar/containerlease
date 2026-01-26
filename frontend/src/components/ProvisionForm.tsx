import React, { useState, useMemo } from 'react'
import { containerApi } from '../services/containerApi'

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
  const [cpuMilli, setCpuMilli] = useState<number>(500)
  const [memoryMB, setMemoryMB] = useState<number>(512)
  const [logDemo, setLogDemo] = useState<boolean>(true)
  const [volumeSizeMB, setVolumeSizeMB] = useState<number>(0)

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

    try {
      const result = await containerApi.provision(imageType, duration, cpuMilli, memoryMB, logDemo, volumeSizeMB > 0 ? volumeSizeMB : undefined)
      
      // Show success toast
      const event = new CustomEvent('toast', {
        detail: {
          type: 'success',
          message: `Container ${result.id.substring(0, 12)} provisioned!`
        }
      })
      window.dispatchEvent(event)

      // Reset form
      setImageType('ubuntu')
      setDuration(30)
      setCpuMilli(500)
      setMemoryMB(512)
      setLogDemo(true)
      setVolumeSizeMB(0)
      onProvisioned?.()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      const event = new CustomEvent('toast', {
        detail: {
          type: 'error',
          message: message
        }
      })
      window.dispatchEvent(event)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: 'flex', gap: '1rem', alignItems: 'center', width: '100%' }}>
      <div className="control-group">
        <label className="control-label">Image</label>
        <select
          value={imageType}
          onChange={(e) => setImageType(e.target.value)}
          disabled={loading}
          className="control-select"
        >
          <option value="ubuntu">Ubuntu 22.04</option>
          <option value="alpine">Alpine Linux</option>
        </select>
      </div>

      <div className="control-group">
        <label className="control-label">Duration</label>
        <input
          type="range"
          min={minDuration}
          max={maxDuration}
          value={duration}
          onChange={(e) => setDuration(parseInt(e.target.value))}
          disabled={loading}
          style={{ width: '120px' }}
        />
        <span style={{ minWidth: '50px', textAlign: 'right', fontFamily: 'monospace', fontSize: '0.9rem' }}>
          {duration}m
        </span>
      </div>

      <div className="control-group">
        <label className="control-label">CPU</label>
        <select
          value={cpuMilli}
          onChange={(e) => setCpuMilli(parseInt(e.target.value))}
          disabled={loading}
          className="control-select"
        >
          <option value="250">250m</option>
          <option value="500">500m</option>
          <option value="1000">1000m</option>
          <option value="2000">2000m</option>
        </select>
      </div>

      <div className="control-group">
        <label className="control-label">Memory</label>
        <select
          value={memoryMB}
          onChange={(e) => setMemoryMB(parseInt(e.target.value))}
          disabled={loading}
          className="control-select"
        >
          <option value="256">256M</option>
          <option value="512">512M</option>
          <option value="1024">1024M</option>
          <option value="2048">2048M</option>
        </select>
      </div>

      <div className="control-group">
        <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.85rem', color: '#8b949e' }}>
          <input
            type="checkbox"
            checked={logDemo}
            onChange={(e) => setLogDemo(e.target.checked)}
            disabled={loading}
            style={{ cursor: 'pointer' }}
          />
          Demo Logs
        </label>
      </div>

      <div style={{ marginLeft: 'auto', display: 'flex', gap: '1rem', alignItems: 'center' }}>
        <span style={{ fontSize: '0.85rem', color: '#6e7681' }}>
          Est: <span style={{ fontFamily: 'monospace', color: '#8b949e' }}>${cost.toFixed(2)}</span>
        </span>
        <button type="submit" disabled={loading} className="btn-provision">
          {loading ? 'Provisioning...' : 'Provision'}
        </button>
      </div>

      {error && (
        <div style={{ position: 'fixed', bottom: '1rem', right: '1rem', background: '#f87171', color: '#fff', padding: '1rem', borderRadius: '8px', fontSize: '0.9rem' }}>
          {error}
        </div>
      )}
    </form>
  )
}
