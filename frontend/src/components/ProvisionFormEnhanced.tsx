import React, { useState } from 'react'
import { containerApi } from '../services/containerApi'
import '../styles/provision-form.css'

interface ProvisionFormProps {
  onSuccess: () => void
  onCancel?: () => void
}

type IsolationLevel = 'strict' | 'standard' | 'permissive'

export const ProvisionFormEnhanced: React.FC<ProvisionFormProps> = ({ onSuccess, onCancel }) => {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  // Form state
  const [imageType, setImageType] = useState('alpine')
  const [duration, setDuration] = useState(30)
  const [cpuMilli, setCpuMilli] = useState(500)
  const [memoryMB, setMemoryMB] = useState(512)
  const [volumeSizeMB, setVolumeSizeMB] = useState(0)
  const [logDemo, setLogDemo] = useState(true)
  const [isolationLevel, setIsolationLevel] = useState<IsolationLevel>('standard')
  const [readOnlyFS, setReadOnlyFS] = useState(true)
  const [networkAccess, setNetworkAccess] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const result = await containerApi.provision(
        imageType,
        duration,
        cpuMilli,
        memoryMB,
        logDemo,
        volumeSizeMB > 0 ? volumeSizeMB : undefined
      )

      setSuccess(true)
      setTimeout(() => {
        onSuccess()
        setSuccess(false)
        // Reset form
        setImageType('alpine')
        setDuration(30)
        setCpuMilli(500)
        setMemoryMB(512)
        setVolumeSizeMB(0)
        setLogDemo(true)
        setError(null)
      }, 1500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to provision container')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="provision-container">
      <div className="provision-panel">
        <div className="provision-header">
          <div>
            <h2>Create Container</h2>
            <p>Provision a temporary, isolated environment</p>
          </div>
          {onCancel && (
            <button className="btn-icon" onClick={onCancel} title="Close">
              ✕
            </button>
          )}
        </div>

        {success && (
          <div className="alert alert-success">
            ✓ Container provisioned successfully!
          </div>
        )}

        {error && (
          <div className="alert alert-critical">
            ⚠ {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="provision-form">
          {/* Image Selection */}
          <fieldset className="provision-section">
            <legend>Image & Environment</legend>
            <div className="form-group">
              <label htmlFor="imageType" className="form-label">Container Image</label>
              <select
                id="imageType"
                className="form-select"
                value={imageType}
                onChange={(e) => setImageType(e.target.value)}
                disabled={loading}
              >
                <option value="alpine">Alpine Linux (minimal, ~5MB)</option>
                <option value="ubuntu">Ubuntu 22.04 (full, ~77MB)</option>
              </select>
              <small>Choose a base image for your container environment</small>
            </div>

            <div className="form-group">
              <label htmlFor="logDemo" className="form-checkbox">
                <input
                  id="logDemo"
                  type="checkbox"
                  checked={logDemo}
                  onChange={(e) => setLogDemo(e.target.checked)}
                  disabled={loading}
                />
                <span>Enable Demo Logs</span>
              </label>
              <small>Generate sample logs for testing log streaming</small>
            </div>
          </fieldset>

          {/* Resource Allocation */}
          <fieldset className="provision-section">
            <legend>Resource Allocation</legend>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="duration" className="form-label">Duration (Minutes)</label>
                <div className="input-with-suffix">
                  <input
                    id="duration"
                    type="range"
                    min="5"
                    max="120"
                    value={duration}
                    onChange={(e) => setDuration(parseInt(e.target.value))}
                    disabled={loading}
                    className="form-range"
                  />
                  <input
                    type="number"
                    min="5"
                    max="120"
                    value={duration}
                    onChange={(e) => setDuration(Math.max(5, Math.min(120, parseInt(e.target.value) || 5)))}
                    disabled={loading}
                    className="form-input form-input-number"
                  />
                  <span className="input-suffix">min</span>
                </div>
                <small>Auto-destroyed after expiration</small>
              </div>

              <div className="form-group">
                <label htmlFor="cpuMilli" className="form-label">CPU Allocation</label>
                <div className="input-with-suffix">
                  <input
                    id="cpuMilli"
                    type="range"
                    min="250"
                    max="2000"
                    step="250"
                    value={cpuMilli}
                    onChange={(e) => setCpuMilli(parseInt(e.target.value))}
                    disabled={loading}
                    className="form-range"
                  />
                  <input
                    type="number"
                    min="250"
                    max="2000"
                    step="250"
                    value={cpuMilli}
                    onChange={(e) => setCpuMilli(Math.max(250, Math.min(2000, parseInt(e.target.value) || 500)))}
                    disabled={loading}
                    className="form-input form-input-number"
                  />
                  <span className="input-suffix">m</span>
                </div>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="memoryMB" className="form-label">Memory</label>
                <div className="input-with-suffix">
                  <input
                    id="memoryMB"
                    type="range"
                    min="256"
                    max="2048"
                    step="256"
                    value={memoryMB}
                    onChange={(e) => setMemoryMB(parseInt(e.target.value))}
                    disabled={loading}
                    className="form-range"
                  />
                  <input
                    type="number"
                    min="256"
                    max="2048"
                    step="256"
                    value={memoryMB}
                    onChange={(e) => setMemoryMB(Math.max(256, Math.min(2048, parseInt(e.target.value) || 512)))}
                    disabled={loading}
                    className="form-input form-input-number"
                  />
                  <span className="input-suffix">MB</span>
                </div>
              </div>

              <div className="form-group">
                <label htmlFor="volumeSizeMB" className="form-label">Storage (Optional)</label>
                <div className="input-with-suffix">
                  <input
                    id="volumeSizeMB"
                    type="number"
                    min="0"
                    max="10240"
                    step="256"
                    value={volumeSizeMB}
                    onChange={(e) => setVolumeSizeMB(Math.max(0, parseInt(e.target.value) || 0))}
                    disabled={loading}
                    className="form-input"
                    placeholder="0"
                  />
                  <span className="input-suffix">MB</span>
                </div>
              </div>
            </div>
          </fieldset>

          {/* Security & Isolation */}
          <fieldset className="provision-section provision-section-security">
            <legend>🔒 Security & Isolation</legend>

            <div className="form-group">
              <label className="form-label">Isolation Level</label>
              <div className="isolation-selector">
                <label className={`isolation-option ${isolationLevel === 'strict' ? 'selected' : ''}`}>
                  <input
                    type="radio"
                    name="isolation"
                    value="strict"
                    checked={isolationLevel === 'strict'}
                    onChange={(e) => setIsolationLevel(e.target.value as IsolationLevel)}
                    disabled={loading}
                  />
                  <div className="isolation-content">
                    <strong>Strict</strong>
                    <small>Read-only FS, no network, no privileges</small>
                  </div>
                </label>

                <label className={`isolation-option ${isolationLevel === 'standard' ? 'selected' : ''}`}>
                  <input
                    type="radio"
                    name="isolation"
                    value="standard"
                    checked={isolationLevel === 'standard'}
                    onChange={(e) => setIsolationLevel(e.target.value as IsolationLevel)}
                    disabled={loading}
                  />
                  <div className="isolation-content">
                    <strong>Standard</strong>
                    <small>Writable FS, isolated networking, no privileges</small>
                  </div>
                </label>

                <label className={`isolation-option ${isolationLevel === 'permissive' ? 'selected' : ''}`}>
                  <input
                    type="radio"
                    name="isolation"
                    value="permissive"
                    checked={isolationLevel === 'permissive'}
                    onChange={(e) => setIsolationLevel(e.target.value as IsolationLevel)}
                    disabled={loading}
                  />
                  <div className="isolation-content">
                    <strong>Permissive</strong>
                    <small>⚠ Full access (risky, only for trusted code)</small>
                  </div>
                </label>
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="readOnlyFS" className="form-checkbox">
                <input
                  id="readOnlyFS"
                  type="checkbox"
                  checked={readOnlyFS}
                  onChange={(e) => setReadOnlyFS(e.target.checked)}
                  disabled={loading || isolationLevel === 'strict'}
                />
                <span>Read-Only Filesystem</span>
              </label>
              <small>Prevents code from modifying the environment</small>
            </div>

            {networkAccess && (
              <div className="alert alert-warning">
                ⚠ <strong>Network Access Enabled:</strong> Container has internet connectivity. Ensure you trust the code you're running.
              </div>
            )}

            <div className="form-group">
              <label htmlFor="networkAccess" className="form-checkbox">
                <input
                  id="networkAccess"
                  type="checkbox"
                  checked={networkAccess}
                  onChange={(e) => setNetworkAccess(e.target.checked)}
                  disabled={loading}
                />
                <span>Enable Network Access</span>
              </label>
              <small>Allows container to make external connections</small>
            </div>
          </fieldset>

          {/* Actions */}
          <div className="provision-actions">
            {onCancel && (
              <button
                type="button"
                className="btn btn-secondary"
                onClick={onCancel}
                disabled={loading}
              >
                Cancel
              </button>
            )}
            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading}
            >
              {loading ? '⟳ Provisioning...' : '▶ Create Container'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
