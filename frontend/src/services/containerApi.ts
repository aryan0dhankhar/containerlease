import type { Container } from '../types/container'

const BACKEND_URL = 'http://localhost:8080'

interface ProvisionResponse {
  id: string
  status: string
  expiryTime: string
  createdAt: string
  imageType: string
  cost: number
}

interface ContainerListResponse {
  containers: Container[]
}

interface ProvisionStatusResponse {
  id: string
  status: string
  imageType: string
  cost: number
  createdAt: string
  expiryTime: string
  error?: string
  timeLeftSeconds: number
}

export const containerApi = {
  async provision(
    imageType: string,
    durationMinutes: number,
    cpuMilli?: number,
    memoryMB?: number,
    logDemo?: boolean
  ): Promise<ProvisionResponse> {
    const response = await fetch(`${BACKEND_URL}/api/provision`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ imageType, durationMinutes, cpuMilli, memoryMB, logDemo }),
    })

    if (!response.ok) {
      throw new Error(`Provision failed: ${response.statusText}`)
    }

    return response.json()
  },

  async getContainers(): Promise<Container[]> {
    const response = await fetch(`${BACKEND_URL}/api/containers`)

    if (!response.ok) {
      throw new Error(`Get containers failed: ${response.statusText}`)
    }

    const data: ContainerListResponse = await response.json()
    return data.containers || []
  },

  async getProvisionStatus(id: string): Promise<ProvisionStatusResponse> {
    const response = await fetch(`${BACKEND_URL}/api/containers/${id}/status`)

    if (!response.ok) {
      throw new Error(`Get status failed: ${response.statusText}`)
    }

    return response.json()
  },

  async deleteContainer(id: string): Promise<void> {
    const response = await fetch(`${BACKEND_URL}/api/containers/${id}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      throw new Error(`Delete failed: ${response.statusText}`)
    }
  },

  subscribeToLogs(
    containerId: string,
    onMessage: (message: string) => void,
    onError: (error: Error) => void
  ): () => void {
    const ws = new WebSocket(`ws://localhost:8080/ws/logs/${containerId}`)

    ws.onmessage = (event) => {
      onMessage(event.data)
    }

    ws.onerror = () => {
      onError(new Error('WebSocket error'))
    }

    return () => {
      ws.close()
    }
  },
}
