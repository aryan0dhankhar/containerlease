// Container represents a container in the API
export interface Container {
  id: string
  imageType: string
  status: string
  cost?: number // Cost in dollars
  createdAt: string
  expiryAt: string // ISO timestamp when container expires
  expiresIn?: number // seconds (deprecated, computed from expiryAt)
  cpuMilli?: number // CPU allocation in millicores
  memoryMB?: number // Memory allocation in MB
}

// ProvisionRequest for creating containers
export interface ProvisionRequest {
  imageType: string
  durationMinutes: number
  cpuMilli?: number
  memoryMB?: number
}

// ProvisionResponse after container creation
export interface ProvisionResponse {
  id: string
  expiryTime: string
  createdAt: string
}

// LogEntry represents a log message from container
export interface LogEntry {
  timestamp: string
  level: string
  message: string
}

// LoginResponse from authentication
export interface LoginResponse {
  token: string
  expiresAt: string
  tenantId: string
  userId: string
}
