import React, { useEffect, useState, useRef } from 'react'
import { containerApi } from '../services/containerApi'
import './ContainerLogs.css'

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
  const logsEndRef = useRef<HTMLDivElement>(null)
  const unsubscribeRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    setConnectionState('connecting')
    setError(null)

    const unsubscribe = containerApi.subscribeToLogs(
      containerId,
      (message: string) => {
        setLogs((prev) => [...prev, message])
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
  }, [containerId])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const getStatusColor = () => {
    switch (connectionState) {
      case 'connecting':
        return 'yellow'
      case 'connected':
        return 'green'
      case 'error':
        return 'red'
      case 'disconnected':
        return 'gray'
      default:
        return 'gray'
    }
  }

  return (
    <div className="container-logs-modal">
      <div className="logs-panel">
        <div className="logs-header">
          <div className="logs-title">
            <h3>Container Logs</h3>
            <code className="container-id-badge">
              {containerId.substring(0, 12)}
            </code>
          </div>
          <div className="logs-controls">
            <span
              className="connection-status"
              style={{ color: getStatusColor() }}
            >
              ● {connectionState}
            </span>
            <button onClick={onClose} className="btn btn-close">
              ×
            </button>
          </div>
        </div>

        <div className="logs-body">
          {error && (
            <div className="logs-error">
              <strong>Error:</strong> {error}
            </div>
          )}
          {connectionState === 'connecting' && (
            <div className="logs-loading">Connecting to log stream...</div>
          )}
          {logs.length === 0 && connectionState === 'connected' && (
            <div className="logs-empty">No logs yet. Waiting for output...</div>
          )}
          <div className="logs-content">
            {logs.map((log, index) => (
              <div key={index} className="log-line">
                {log}
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        </div>
      </div>
    </div>
  )
}
