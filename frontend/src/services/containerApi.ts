import type { Container } from '../types/container'

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080'

export interface Preset {
  id: string
  name: string
  cpuMilli: number
  memoryMB: number
  durationMin: number
}

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

interface PresetsResponse {
  presets: Preset[]
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

export interface LoginResponse {
  token: string
  expiresAt: string
  tenantId: string
  userId: string
}

export const containerApi = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await fetch(`${BACKEND_URL}/api/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Login failed' }))
      throw new Error(error.error || 'Login failed')
    }

    return response.json()
  },

  async getPresets(): Promise<Preset[]> {
    const response = await fetch(`${BACKEND_URL}/api/presets`)
    if (!response.ok) {
      throw new Error(`Get presets failed: ${response.statusText}`)
    }
    const data: PresetsResponse = await response.json()
    return data.presets || []
  },

  async provision(
    imageType: string,
    durationMinutes: number,
    cpuMilli?: number,
    memoryMB?: number,
    logDemo?: boolean,
    volumeSizeMB?: number
  ): Promise<ProvisionResponse> {
    const token = localStorage.getItem('token')
    const response = await fetch(`${BACKEND_URL}/api/provision`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
      },
      body: JSON.stringify({ imageType, durationMinutes, cpuMilli, memoryMB, logDemo, volumeSizeMB }),
    })

    if (response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
      throw new Error('Session expired, please login again')
    }

    if (!response.ok) {
      throw new Error(`Provision failed: ${response.statusText}`)
    }

    return response.json()
  },

  async getContainers(): Promise<Container[]> {
    const token = localStorage.getItem('token')
    const response = await fetch(`${BACKEND_URL}/api/containers`, {
      headers: token ? { 'Authorization': `Bearer ${token}` } : {},
    })

    if (response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
      throw new Error('Session expired, please login again')
    }

    if (!response.ok) {
      throw new Error(`Get containers failed: ${response.statusText}`)
    }

    const data: ContainerListResponse = await response.json()
    return data.containers || []
  },

  async getProvisionStatus(id: string): Promise<ProvisionStatusResponse> {
    const token = localStorage.getItem('token')
    const response = await fetch(`${BACKEND_URL}/api/containers/${id}/status`, {
      headers: token ? { 'Authorization': `Bearer ${token}` } : {},
    })

    if (response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
      throw new Error('Session expired, please login again')
    }

    if (!response.ok) {
      throw new Error(`Get status failed: ${response.statusText}`)
    }

    return response.json()
  },

  async deleteContainer(id: string): Promise<void> {
    const token = localStorage.getItem('token')
    const response = await fetch(`${BACKEND_URL}/api/containers/${id}`, {
      method: 'DELETE',
      headers: token ? { 'Authorization': `Bearer ${token}` } : {},
    })

    if (response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
      throw new Error('Session expired, please login again')
    }

    if (!response.ok) {
      throw new Error(`Delete failed: ${response.statusText}`)
    }
  },

  subscribeToLogs(
    containerId: string,
    onMessage: (message: string) => void,
    onError: (error: Error) => void
  ): () => void {
    const wsUrl = BACKEND_URL.replace('http://', 'ws://').replace('https://', 'wss://')
    const ws = new WebSocket(`${wsUrl}/ws/logs/${containerId}`)

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = (event) => {
      onMessage(event.data)
    }

    ws.onerror = (event) => {
      console.error('WebSocket error:', event)
      onError(new Error('WebSocket connection failed'))
    }

    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason)
    }

    return () => {
      ws.close()
    }
  },
}
