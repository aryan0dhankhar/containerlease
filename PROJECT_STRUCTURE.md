# ContainerLease - Project Structure Summary

## Directory Tree

```
containerlease/
├── README.md                          # Main documentation
├── ARCHITECTURE.md                    # Detailed architecture & cleanup logic
├── docker-compose.yml                 # Local dev environment
├── .gitignore
│
├── backend/                           # Go backend service
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # ⭐ Application entry point
│   │
│   ├── internal/                      # Private implementation
│   │   ├── domain/
│   │   │   └── container.go          # Domain entities & interfaces
│   │   │
│   │   ├── handler/                   # HTTP request handlers (Layer 1)
│   │   │   ├── provision.go          # POST /api/provision
│   │   │   ├── logs.go               # WS /ws/logs/{id}
│   │   │   └── status.go             # GET /api/containers
│   │   │
│   │   ├── service/                   # Business logic (Layer 2)
│   │   │   ├── container_service.go  # Provisioning orchestration
│   │   │   └── lifecycle_service.go  # Container lifecycle
│   │   │
│   │   ├── repository/                # Data access (Layer 3)
│   │   │   ├── lease_repository.go   # Redis lease storage
│   │   │   └── container_repository.go
│   │   │
│   │   ├── infrastructure/            # External clients (Layer 4)
│   │   │   ├── docker/
│   │   │   │   ├── client.go        # Docker SDK wrapper
│   │   │   │   └── container.go     # Docker operations
│   │   │   ├── redis/
│   │   │   │   └── client.go        # Redis client wrapper
│   │   │   └── logger/
│   │   │       └── logger.go        # Structured logging (slog)
│   │   │
│   │   ├── middleware/                # HTTP middleware
│   │   │   ├── error_handler.go
│   │   │   └── request_logger.go
│   │   │
│   │   └── worker/
│   │       └── cleanup_worker.go      # ⭐⭐⭐ CLEANUP LOGIC (runs every 1 min)
│   │
│   ├── pkg/                           # Reusable packages
│   │   ├── config/
│   │   │   └── config.go             # Configuration management
│   │   ├── errs/
│   │   │   └── errors.go             # Custom error types
│   │   └── dto/
│   │       └── container.go
│   │
│   ├── config/
│   │   └── .env.example              # Environment variable template
│   │
│   ├── go.mod                         # Go dependencies
│   ├── go.sum
│   └── Dockerfile                     # Container build
│
├── frontend/                          # React + TypeScript frontend
│   ├── src/
│   │   ├── components/                # React components
│   │   │   ├── ProvisionForm.tsx     # "Provision Container" UI
│   │   │   ├── ContainerList.tsx     # List of containers
│   │   │   ├── LogViewer.tsx         # Real-time logs
│   │   │   └── ExpiryTimer.tsx       # Countdown display
│   │   │
│   │   ├── hooks/                     # Custom React hooks
│   │   │   ├── useContainers.ts      # Container state
│   │   │   ├── useWebSocket.ts       # WebSocket management
│   │   │   └── useTimer.ts           # Timer logic
│   │   │
│   │   ├── services/                  # API integration
│   │   │   ├── containerApi.ts       # REST + WebSocket calls
│   │   │   └── logService.ts         # Log streaming
│   │   │
│   │   ├── types/                     # TypeScript types
│   │   │   ├── container.ts
│   │   │   └── api.ts
│   │   │
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── index.css
│   │
│   ├── public/
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── package.json                   # npm dependencies
│   └── Dockerfile
```

## Where Does Docker Cleanup Logic Live?

### 🎯 Primary Location: `backend/internal/worker/cleanup_worker.go`

This file contains the **automatic garbage collector** that:

1. **Runs independently** in a background goroutine (started by main.go)
2. **Every 1 minute** (configurable), queries Redis for expired leases
3. **For each expired container:**
   - Stops the Docker container
   - Removes the Docker container
   - Deletes from container repository
   - Deletes lease from Redis
4. **Implements retry logic** with exponential backoff
5. **Logs all operations** with structured context (slog)

### Why This Design?

**Without a worker:**
- Containers only get cleaned up if manually deleted
- If no one deletes them → they run forever
- Manual cleanup is not guaranteed

**With a background worker:**
- Automatic cleanup every 1 minute
- No dependency on user action
- Time-bound container lifetime is enforced
- Guaranteed resource cleanup

### Key Code Section

```go
// In cleanup_worker.go:
func (w *CleanupWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)  // Every 1 minute
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.cleanupExpiredContainers(ctx)  // Check & clean
        }
    }
}
```

## Clean Architecture Layers

### Layer 1: Handlers (HTTP/WebSocket)
- **File:** `internal/handler/*`
- **Responsibility:** Parse requests, call services, return responses
- **Example:** `ProvisionHandler` receives POST /api/provision

### Layer 2: Services (Business Logic)
- **File:** `internal/service/*`
- **Responsibility:** Orchestrate domain logic, coordinate repositories
- **Example:** `ContainerService.ProvisionContainer()` creates container + lease

### Layer 3: Repositories (Data Access)
- **File:** `internal/repository/*`
- **Responsibility:** Abstract persistence (Redis, filesystem)
- **Example:** `LeaseRepository` stores/retrieves leases from Redis

### Layer 4: Infrastructure (External APIs)
- **File:** `internal/infrastructure/*`
- **Responsibility:** Wrap external clients (Docker, Redis)
- **Example:** `docker/client.go` wraps Docker SDK

### Special: Worker Layer (Background Jobs)
- **File:** `internal/worker/*`
- **Responsibility:** Periodic/async tasks
- **Example:** `cleanup_worker.go` runs every 1 minute

## Tech Stack Summary

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Backend** | Go 1.21+ | Efficient concurrency, Docker SDK |
| **Frontend** | React 18 + TypeScript | Type-safe UI components |
| **Build Tool** | Vite | Fast frontend bundling |
| **State Management** | Redis | Session + lease storage with TTL |
| **Container Runtime** | Docker (native SDK) | Provision & lifecycle management |
| **Logging** | slog (stdlib) | Structured JSON logging |
| **Communication** | REST + WebSocket | API + real-time logs |
| **Dev Environment** | Docker Compose | Orchestrate Redis + Backend + Frontend |

## Quick Start Files

### Configuration
- `backend/config/.env.example` - All configurable variables

### Docker Compose
- `docker-compose.yml` - Single command to spin up entire stack

### Key Implementation Files
- `backend/cmd/server/main.go` - Entry point
- `backend/internal/worker/cleanup_worker.go` - **Cleanup logic**
- `backend/internal/service/container_service.go` - Provisioning logic
- `frontend/src/components/ProvisionForm.tsx` - UI for provisioning

## API Contract

### REST Endpoints
```
POST /api/provision
  Body: { imageType: string, durationMinutes: number }
  Response: { id: string, expiryTime: string, createdAt: string }

GET /api/containers
  Response: { containers: Container[] }

DELETE /api/containers/{id}
  Response: { success: boolean }
```

### WebSocket Endpoints
```
WS /ws/logs/{containerId}
  Streams: { timestamp: string, level: string, message: string }
```

## Development Workflow

### 1. Setup
```bash
cd containerlease
docker-compose up -d  # Start Redis, Backend, Frontend
```

### 2. Backend Development
```bash
cd backend
go run ./cmd/server  # Runs with hot reload capability
```

### 3. Frontend Development
```bash
cd frontend
npm run dev  # Starts Vite dev server
```

### 4. Testing Cleanup
```bash
# Provision a 5-minute container
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"ubuntu","durationMinutes":5}'

# Wait ~5 minutes, check cleanup worker logs
docker logs containerlease-backend-1 | grep cleanup
```

## Next Implementation Steps

1. **Implement repositories** (Redis integration)
2. **Implement Docker client** (container creation/removal)
3. **Implement remaining handlers** (logs, status, list)
4. **Add tests** for cleanup logic (critical!)
5. **Add integration tests** with real Docker
6. **Frontend components** (connect to API)
7. **Deployment** (Kubernetes/Docker configs)

---

**See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed cleanup logic explanation.**
