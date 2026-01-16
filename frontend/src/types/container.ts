// Container represents a container in the API
export interface Container {
  id: string
  imageType: string
  status: string
  createdAt: string
  expiryTime: string
  expiresIn: number // seconds
}

// ProvisionRequest for creating containers
export interface ProvisionRequest {
  imageType: string
  durationMinutes: number
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
