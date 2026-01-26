import React, { useEffect, useState, useRef } from 'react'
import { containerApi } from '../services/containerApi'

interface ContainerLogsProps {
  containerId: string
  onClose: () => void
}

type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error'

export const ContainerLogs: React.FC<ContainerLogsProps> = ({
  containerId,
  onClose,
}) => {
  const [logs, setLogs] = useState<string[]>([])
  const [connectionState, setConnectionState] =
    useState<ConnectionState>('connecting')
  const [error, setError] = useState<string | null>(null)
  const [isPaused, setIsPaused] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const unsubscribeRef = useRef<(() => void) | null>(null)
  const pendingLogsRef = useRef<string[]>([])

  useEffect(() => {
    setConnectionState('connecting')
    setError(null)

    const unsubscribe = containerApi.subscribeToLogs(
      containerId,
      (message: string) => {
        if (isPaused) {
          pendingLogsRef.current.push(message)
        } else {
          setLogs((prev) => [...prev, message])
        }
        setConnectionState('connected')
      },
      (err: Error) => {
        setError(err.message)
        setConnectionState('error')
      }
    )

    unsubscribeRef.current = unsubscribe

    return () => {
      if (unsubscribeRef.current) {
        unsubscribeRef.current()
      }
    }
  }, [containerId, isPaused])

  useEffect(() => {
    if (!isPaused) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, isPaused])

  const handleResume = () => {
    if (pendingLogsRef.current.length > 0) {
      setLogs((prev) => [...prev, ...pendingLogsRef.current])
      pendingLogsRef.current = []
    }
    setIsPaused(false)
  }

  const handleDownload = () => {
    const content = logs.join('\n')
    const element = document.createElement('a')
    element.setAttribute(
      'href',
      'data:text/plain;charset=utf-8,' + encodeURIComponent(content)
    )
    element.setAttribute('download', `logs-${containerId.substring(0, 12)}.txt`)
    element.style.display = 'none'
    document.body.appendChild(element)
    element.click()
    document.body.removeChild(element)
  }

  return (
    <div className="logs-modal">
      <div className="logs-container">
        <div className="logs-header">
          <div className="logs-title">
            Terminal · {containerId.substring(0, 12)}
          </div>
          <button onClick={onClose} className="logs-close">
            ✕
          </button>
        </div>

        <div className="logs-viewer">
          {connectionState === 'connecting' && (
            <div className="logs-loading">$ Connecting to log stream...</div>
          )}
          {error && (
            <div style={{ color: '#f87171' }}>
              Error: {error}
            </div>
          )}
          {logs.length === 0 && connectionState === 'connected' && (
            <div className="logs-empty">$ Waiting for output...</div>
          )}
          {logs.map((log, index) => (
            <div key={index}>
              {log}
            </div>
          ))}
          {isPaused && (
            <div style={{ color: '#facc15', marginTop: '1rem' }}>
              $ [PAUSED - {pendingLogsRef.current.length} pending]
            </div>
          )}
          <div ref={logsEndRef} />
        </div>

        <div style={{ borderTop: '1px solid var(--border)', padding: '1rem', display: 'flex', gap: '0.5rem' }}>
          <button
            onClick={() => (isPaused ? handleResume() : setIsPaused(true))}
            className="btn"
            style={{ color: isPaused ? 'var(--accent-amber)' : 'var(--accent-blue)', borderColor: isPaused ? 'var(--accent-amber)' : 'var(--accent-blue)' }}
          >
            {isPaused ? `Resume` : 'Pause'}
          </button>
          <button onClick={handleDownload} className="btn" style={{ color: 'var(--accent-green)', borderColor: 'var(--accent-green)' }}>
            Download
          </button>
          <button
            onClick={() => setLogs([])}
            className="btn"
            style={{ color: 'var(--accent-red)', borderColor: 'var(--accent-red)' }}
          >
            Clear
          </button>
        </div>
      </div>
    </div>
  )
}
