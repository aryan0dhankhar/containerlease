# ContainerLease System Architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Provision    │  │ Container    │  │  Log Viewer  │          │
│  │ Form         │  │ List         │  │  (WebSocket) │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP/WebSocket
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Backend (Go + Gorilla)                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  HTTP Handlers (provision, status, delete, logs WS)       │ │
│  └────────────┬───────────────────────────────────────────────┘ │
│               │                                                  │
│  ┌────────────┴───────────────────────────────────────────────┐ │
│  │  Container Service (provisioning, lifecycle, billing)     │ │
│  └────────────┬───────────────────────────────────────────────┘ │
│               │                                                  │
│  ┌────────────┴────────────┬──────────────────────────────────┐ │
│  │  Lease Repository       │  Container Repository           │ │
│  │  (Redis TTL)            │  (Redis with TTL)               │ │
│  └─────────────────────────┴──────────────────────────────────┘ │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Cleanup Worker (GC, reconciliation, billing)           │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────┬──────────────────────┬────────────────────────┘
                 │                      │
                 ↓                      ↓
        ┌────────────────┐    ┌────────────────┐
        │ Docker Engine  │    │ Redis (State)  │
        │ (Containers)   │    │ (Leases/Meta)  │
        └────────────────┘    └────────────────┘
```

## Request Flow

### 1. Container Provisioning
```
User → POST /api/provision
  ↓
CORS Middleware → Request ID Middleware
  ↓
Provision Handler
  ├─ Validate: image allowlist, duration bounds, resource caps
  ├─ Apply defaults: CPU/memory if not specified
  └─ Pass to Service
       ↓
Container Service
  ├─ Generate domain ID
  ├─ Create container record (status: pending)
  ├─ Store in Redis with TTL
  ├─ Create lease with TTL in Redis
  └─ Launch async Docker provisioning goroutine
       ↓
Docker Client (async)
  ├─ Pull image (if needed)
  ├─ Create container with resource limits
  ├─ Start container
  ├─ Update status to "running" + Docker ID
  └─ Handle errors → status "error"
```

### 2. Log Streaming
```
User → WebSocket ws://host/ws/logs/{id}
  ↓
CORS/Origin Check
  ↓
Logs Handler
  ├─ Lookup container by domain ID
  ├─ Resolve Docker ID from metadata
  ├─ Start heartbeat ping goroutine (15s interval)
  └─ Stream Docker logs via WebSocket
       ↓
Docker Client
  └─ ContainerLogs(dockerID, follow=true)
       ↓
User receives real-time log lines
```

### 3. Garbage Collection Cycle
```
Every N minutes (configurable):
  ↓
Cleanup Worker
  ├─ List all containers from Redis
  ├─ For each container:
  │   ├─ Check if expired (now > expiryAt)
  │   ├─ Check if orphaned (missing lease)
  │   └─ If yes → terminate
  └─ Terminate flow:
       ├─ Fetch container metadata
       ├─ If dockerID exists:
       │   ├─ Stop Docker container
       │   └─ Remove Docker container
       ├─ Calculate final cost (runtime * hourly rate)
       ├─ Update status → "terminated"
       ├─ Set new TTL (15min for billing visibility)
       └─ Delete lease from Redis
```

## Data Models

### Container (Domain)
```go
type Container struct {
    ID        string    // Domain ID (e.g., container-1234567890)
    DockerID  string    // Actual Docker container ID
    ImageType string    // ubuntu, alpine
    Status    string    // pending, running, error, terminated
    CPUMilli  int       // Requested CPU (millicores)
    MemoryMB  int       // Requested memory (MB)
    CreatedAt time.Time
    ExpiryAt  time.Time // Lease expiration
    Cost      float64   // Final cost (updated on termination)
    Error     string    // Error message if status=error
}
```

### Lease (Domain)
```go
type Lease struct {
    ContainerID     string
    LeaseKey        string    // lease:{containerID}
    ExpiryTime      time.Time
    DurationMinutes int
    CreatedAt       time.Time
}
```

### Redis Storage
```
Key: container:{id}
Value: JSON-encoded Container
TTL: time.Until(ExpiryAt) OR 15min (if terminated)

Key: lease:{id}
Value: JSON-encoded Lease
TTL: time.Until(ExpiryTime)
```

## Container States

```
┌─────────┐
│ Pending │ ← Metadata created, Docker container starting
└────┬────┘
     ↓ (Docker create success)
┌─────────┐
│ Running │ ← Container active, logs available
└────┬────┘
     ↓ (Expired OR manually deleted)
┌────────────┐
│ Terminated │ ← Container stopped, final cost calculated
└────┬───────┘
     ↓ (15min retention expires)
┌─────────┐
│ Deleted │ ← Metadata removed from Redis
└─────────┘

   OR (Docker create failed)
┌─────────┐
│  Error  │ ← Provisioning failed, error message stored
└─────────┘
```

## Concurrency & Background Jobs

### Async Provisioning
- Each provision request spawns a goroutine
- Goroutine creates Docker container
- Updates container status atomically via repository
- Errors are captured and stored in container metadata

### WebSocket Heartbeat
- Each log stream spawns a ping goroutine
- Sends ping every 15 seconds
- Terminated when log stream ends

### Cleanup Worker
- Single goroutine running on interval
- Processes all containers sequentially
- Retries failed cleanups with exponential backoff
- Logs all reconciliation actions

## Security Model

### Input Validation
- **Image allowlist**: Only approved images can be provisioned
- **Duration bounds**: 5-120 minutes (configurable)
- **Resource caps**: CPU ≤ 2000m, Memory ≤ 2048MB
- **Request sanitization**: All inputs validated before processing

### CORS & Origin Enforcement
- HTTP endpoints: CORS middleware checks `Origin` header
- WebSocket: Origin validated via `CheckOrigin` function
- Allowed origins configured via environment variable

### Resource Isolation
- Docker containers have hard CPU/memory limits
- No privileged mode or host networking
- Containers run with default isolation

## Observability

### Structured Logging
All logs are JSON-formatted with fields:
- `timestamp`: ISO8601
- `level`: debug, info, warn, error
- `request_id`: UUID for request correlation
- `container_id`: Domain container ID
- `docker_id`: Docker container ID (when applicable)
- `duration_ms`: Request duration
- `error`: Error message (if applicable)

### Request Tracking
- Middleware injects unique request ID
- ID propagated to all log statements
- Returned in `X-Request-ID` header

### Health & Readiness
- `/healthz`: Basic liveness check
- `/readyz`: Redis connectivity check

## Billing Model

### Cost Calculation
```
Hourly Rates:
- Ubuntu: $0.04/hour
- Alpine: $0.01/hour

Initial Cost (provision):
  cost = (durationMinutes / 60) * hourlyRate

Final Cost (termination):
  actualRuntime = now - createdAt
  cost = (actualRuntime.Minutes() / 60) * hourlyRate
```

### Billing Events
1. **Provision**: Estimated cost calculated
2. **Termination**: Final cost calculated based on actual runtime
3. **Record Retention**: Terminated containers kept 15min for billing queries

## Scalability Considerations

### Current Limitations
- Single backend instance (no horizontal scaling yet)
- Redis as single point of failure
- No container placement or node scheduling

### Future Improvements
- Add container orchestration (Kubernetes)
- Implement distributed locking for cleanup worker
- Add metrics export (Prometheus)
- Support multiple backend replicas with leader election
- Add database for persistent billing records
- Implement rate limiting per user/IP
