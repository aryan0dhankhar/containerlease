# ContainerLease API Documentation

## Base URL
`http://localhost:8080`

## Endpoints

### Health & Readiness

#### `GET /healthz`
Health check endpoint.

**Response:**
- `200 OK`: Server is running
- Body: `ok`

#### `GET /readyz`
Readiness check with Redis connectivity test.

**Response:**
- `200 OK`: Server and Redis are ready
- `503 Service Unavailable`: Redis is not ready

---

### Container Provisioning

#### `POST /api/provision`
Provision a new container with specified resources and duration.

**Request Body:**
```json
{
  "imageType": "ubuntu",
  "durationMinutes": 30,
  "cpuMilli": 500,
  "memoryMB": 512,
  "logDemo": true
}
```

**Fields:**
- `imageType` (string, required): Container image type. Must be in allowed list (default: `ubuntu`, `alpine`)
- `durationMinutes` (int, required): Lease duration in minutes. Range: 5-120 (configurable)
- `cpuMilli` (int, optional): CPU allocation in millicores. Default: 500, Max: 2000
- `memoryMB` (int, optional): Memory allocation in MB. Default: 512, Max: 2048
- `logDemo` (bool, optional): Enable demo log output for testing. Default: false

**Response:**
```json
{
  "id": "container-1234567890",
  "status": "pending",
  "expiryTime": "2026-01-25T13:30:00Z",
  "createdAt": "2026-01-25T13:00:00Z",
  "imageType": "ubuntu"
}
```

**Status Codes:**
- `201 Created`: Container provisioned successfully
- `400 Bad Request`: Invalid input (image not allowed, duration out of range, resources exceed limits)
- `500 Internal Server Error`: Provisioning failed

---

### Container Management

#### `GET /api/containers`
List all active containers.

**Response:**
```json
{
  "containers": [
    {
      "id": "container-1234567890",
      "imageType": "ubuntu",
      "status": "running",
      "cpuMilli": 500,
      "memoryMB": 512,
      "createdAt": "2026-01-25T13:00:00Z",
      "expiryAt": "2026-01-25T13:30:00Z",
      "expiresIn": 1800
    }
  ]
}
```

**Status Values:**
- `pending`: Container is being provisioned
- `running`: Container is active
- `error`: Provisioning failed
- `terminated`: Container has been stopped

#### `GET /api/containers/{id}/status`
Get detailed status of a specific container.

**Response:**
```json
{
  "id": "container-1234567890",
  "status": "running",
  "imageType": "ubuntu",
  "createdAt": "2026-01-25T13:00:00Z",
  "expiryTime": "2026-01-25T13:30:00Z",
  "timeLeftSeconds": 1800,
  "error": ""
}
```

#### `DELETE /api/containers/{id}`
Manually terminate a container before its lease expires.

**Response:**
- `204 No Content`: Container terminated successfully
- `404 Not Found`: Container not found
- `500 Internal Server Error`: Termination failed

**Note:** Container is terminated immediately.

---

### Container Logs

#### `GET /ws/logs/{id}`
WebSocket endpoint for streaming container logs in real-time.

**Protocol:** WebSocket (`ws://`)

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/logs/container-1234567890');

ws.onmessage = (event) => {
  console.log('Log:', event.data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

**Requirements:**
- Container must be in `running` status
- Valid Origin header (must match CORS allowed origins)

**Messages:**
- Text frames containing log lines
- Ping/Pong frames for connection keepalive (every 15s)

**Error Messages:**
- `"Error: container not found"`: Container ID doesn't exist
- `"Error: container not yet running"`: Container is still pending
- `"Error: <docker error>"`: Docker daemon error

---

## Resource Limits

### CPU Allocation
- **Unit:** Millicores (1000m = 1 CPU core)
- **Default:** 500m (0.5 CPU)
- **Maximum:** 2000m (2 CPUs)
- **Options:** 250m, 500m, 1000m, 2000m

### Memory Allocation
- **Unit:** Megabytes (MB)
- **Default:** 512 MB
- **Maximum:** 2048 MB (2 GB)
- **Options:** 256 MB, 512 MB, 1024 MB, 2048 MB

### Duration
- **Minimum:** 5 minutes
- **Maximum:** 120 minutes
- **Default:** 30 minutes

---

## Error Handling

### Common Error Responses

**400 Bad Request:**
```json
{
  "error": "imageType not allowed"
}
```

**404 Not Found:**
```json
{
  "error": "container not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "failed to provision container"
}
```

---

## Headers

### CORS
The API supports CORS for allowed origins configured via `CORS_ALLOWED_ORIGINS`.

**Default Allowed Origins:**
- `http://localhost:3000`
- `http://localhost:5173`

### Request Tracking
All responses include:
```
X-Request-ID: <unique-request-id>
```

Use this ID for troubleshooting and log correlation.

---

## Lifecycle Management

### Automatic Cleanup
A background garbage collector runs every minute (configurable) to:
1. Terminate containers past their lease expiry
2. Remove orphaned containers (missing Redis metadata)
3. Clean up containers with missing Docker instances
4. Finalize billing for terminated containers

### Container States Flow
```
provision request
    ↓
pending (metadata created, Docker container starting)
    ↓
running (container active, logs available)
    ↓
terminated (stopped, final cost calculated, metadata retained 15min)
    ↓
deleted (cleanup removes all metadata)
```

### Error State
If provisioning fails:
```
pending → error (metadata updated with error message)
```
