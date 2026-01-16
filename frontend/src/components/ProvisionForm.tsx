import React, { useState } from 'react'
import { containerApi } from '../services/containerApi'
import './ProvisionForm.css'

interface ProvisionFormProps {
  onProvisioned?: () => void
}

export const ProvisionForm: React.FC<ProvisionFormProps> = ({
  onProvisioned,
}) => {
  const [imageType, setImageType] = useState<string>('ubuntu')
  const [duration, setDuration] = useState<number>(30)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    setSuccess(null)

    try {
      const result = await containerApi.provision(imageType, duration)
      setSuccess(
        `Container ${result.id.substring(0, 12)} provisioned successfully!`
      )
      setImageType('ubuntu')
      setDuration(30)
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
          min="5"
          max="480"
          value={duration}
          onChange={(e) => setDuration(parseInt(e.target.value))}
          disabled={loading}
          className="form-input"
        />
        <small>Min: 5 minutes, Max: 8 hours</small>
      </div>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      <button type="submit" disabled={loading} className="btn btn-primary">
        {loading ? 'Provisioning...' : 'Provision Container'}
      </button>
    </form>
  )
}
