# ContainerLease - Complete Project Summary

## 🎯 Project Goal

A web platform where developers click a button to provision temporary Docker containers (Ubuntu/Alpine) for fixed durations (e.g., 2 hours). After expiry, a **background worker automatically kills and removes the container** to free resources.

## 📊 Project Structure Overview

```
containerlease/                       # Monorepo root
├── 📄 README.md                      # Main documentation
├── 📄 ARCHITECTURE.md                # Detailed cleanup logic explanation
├── 📄 PROJECT_STRUCTURE.md           # Directory tree & layer explanation
├── 📄 IMPLEMENTATION_GUIDE.md        # Step-by-step implementation roadmap
├── 📄 docker-compose.yml             # Local dev environment (Redis + Backend + Frontend)
├── 📄 .gitignore
│
├── backend/                          # Go backend (1.21+)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # ⭐ Entry point - initializes worker
│   │
│   ├── internal/                     # Clean Architecture layers
│   │   ├── domain/
│   │   │   └── container.go          # Domain entities & interfaces
│   │   │
│   │   ├── handler/                  # Layer 1: HTTP handlers
│   │   │   ├── provision.go          # POST /api/provision
│   │   │   ├── logs.go               # WS /ws/logs/{id}
│   │   │   └── status.go             # GET /api/containers
│   │   │
│   │   ├── service/                  # Layer 2: Business logic
│   │   │   ├── container_service.go  # Provisioning orchestration
│   │   │   └── lifecycle_service.go  # Lifecycle management
│   │   │
│   │   ├── repository/               # Layer 3: Data access
│   │   │   ├── lease_repository.go   # Redis operations
│   │   │   └── container_repository.go
│   │   │
│   │   ├── infrastructure/           # Layer 4: External clients
│   │   │   ├── docker/
│   │   │   │   ├── client.go         # Docker SDK wrapper
│   │   │   │   └── container.go
│   │   │   ├── redis/
│   │   │   │   └── client.go         # Redis wrapper
│   │   │   └── logger/
│   │   │       └── logger.go         # Structured logging
│   │   │
│   │   ├── middleware/
│   │   │   ├── error_handler.go
│   │   │   └── request_logger.go
│   │   │
│   │   └── worker/                   # ⭐⭐⭐ CLEANUP WORKER
│   │       └── cleanup_worker.go     # Runs every 1 min, checks Redis for expired leases
│   │
│   ├── pkg/
│   │   ├── config/
│   │   │   └── config.go             # Environment configuration
│   │   ├── errs/
│   │   │   └── errors.go             # Custom error types
│   │   └── dto/
│   │       └── container.go
│   │
│   ├── config/
│   │   └── .env.example              # Environment template
│   │
│   ├── go.mod                        # Dependencies: docker, redis, gorilla/websocket
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                         # React + TypeScript + Vite
│   ├── src/
│   │   ├── components/
│   │   │   ├── ProvisionForm.tsx     # Provision UI form
│   │   │   ├── ContainerList.tsx     # List active containers
│   │   │   ├── LogViewer.tsx         # Real-time logs (WebSocket)
│   │   │   └── ExpiryTimer.tsx       # Countdown display
│   │   │
│   │   ├── hooks/
│   │   │   ├── useContainers.ts      # Container state management
│   │   │   ├── useWebSocket.ts       # WebSocket management
│   │   │   └── useTimer.ts           # Timer logic
│   │   │
│   │   ├── services/
│   │   │   ├── containerApi.ts       # REST + WebSocket client
│   │   │   └── logService.ts         # Log streaming
│   │   │
│   │   ├── types/
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
│   ├── package.json
│   └── Dockerfile
```

## 🔴 Critical Component: Cleanup Worker

### Location: `backend/internal/worker/cleanup_worker.go`

This is the **most important piece** of ContainerLease. It ensures containers don't run forever.

### How It Works

1. **Runs continuously** in a background goroutine (started by `main.go`)
2. **Every 1 minute** (configurable), it:
   - Queries Redis for all leases with expired TTL
   - Gets list of container IDs that should be deleted
3. **For each expired container:**
   - Stops the Docker container
   - Removes the Docker container
   - Deletes from repositories
   - Logs the cleanup
4. **Implements retry logic** if cleanup fails (up to 3 retries with exponential backoff)

### Code Structure

```go
// In main.go:
cleanupWorker := worker.NewCleanupWorker(...)
go cleanupWorker.Start(ctx)  // Starts background loop

// In cleanup_worker.go:
func (w *CleanupWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)  // Every 1 min
    for {
        select {
        case <-ticker.C:
            w.cleanupExpiredContainers(ctx)  // Check & clean
        }
    }
}
```

## 🏗️ Clean Architecture Layers

### Layer 1: Handlers (HTTP/WebSocket)
- **Files:** `internal/handler/*`
- Receives requests, calls services, returns responses
- No business logic

### Layer 2: Services (Business Logic)
- **Files:** `internal/service/*`
- Coordinates repositories and domain logic
- Contains provisioning & lifecycle logic

### Layer 3: Repositories (Data Access)
- **Files:** `internal/repository/*`
- Abstracts data persistence (Redis)
- No direct service usage

### Layer 4: Infrastructure (External Clients)
- **Files:** `internal/infrastructure/*`
- Wraps Docker SDK, Redis client, logger
- No business logic

### Special: Worker (Background Jobs)
- **Files:** `internal/worker/*`
- Runs periodically or triggered
- The cleanup worker is here!

## 💡 Why This Architecture?

```
❌ Without background worker:
  - Container cleanup only on user action
  - If no one calls delete → container runs forever
  - Unpredictable resource usage

✅ With background worker:
  - Automatic cleanup every 1 minute
  - Guaranteed time-bound container lifetime
  - Predictable resource cleanup
  - Independent of client behavior
```

## 🚀 Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.21+ |
| Frontend | React 18 + TypeScript + Vite |
| State/Cache | Redis |
| Container Runtime | Docker (native SDK) |
| Communication | REST API + WebSocket (Gorilla) |
| Logging | slog (structured JSON) |
| Infrastructure | Docker Compose (dev), Docker (prod) |

## 📋 API Contract

### REST Endpoints

```
POST /api/provision
  Request: { imageType: "ubuntu"|"alpine"|"debian", durationMinutes: 5-120 }
  Response: { id: string, expiryTime: string, createdAt: string }
  Status: 201 Created

GET /api/containers
  Response: { containers: Container[] }
  Status: 200 OK

DELETE /api/containers/{id}
  Response: { success: boolean }
  Status: 200 OK
```

### WebSocket Endpoints

```
WS /ws/logs/{containerId}
  Sends: { timestamp: string, level: string, message: string }
  For real-time container log streaming
```

## 📝 Data Models

### Container (Domain Entity)
```go
type Container struct {
    ID        string    // Docker container ID
    ImageType string    // "ubuntu", "alpine", "debian"
    Status    string    // "running", "exited", "stopped"
    CreatedAt time.Time
    ExpiryAt  time.Time
}
```

### Lease (Time-bound Reservation)
```go
type Lease struct {
    ContainerID     string
    LeaseKey        string    // "lease:abc123"
    ExpiryTime      time.Time
    DurationMinutes int
    CreatedAt       time.Time
}
```

## 🔄 Request Flow Example

### User Provisions Container (120 min)

```
Frontend:
  Click "Provision Container"
  imageType: "ubuntu"
  duration: 120 minutes
    ↓
Backend:
  POST /api/provision
    ↓
  ProvisionHandler.ServeHTTP()
    ↓
  ContainerService.ProvisionContainer()
    ├─ Docker: Create container → "abc123"
    ├─ Repository: Save container details
    └─ Redis: Store lease "lease:abc123" with TTL=7200s
    ↓
  Response: {
    id: "abc123",
    expiryTime: "2025-01-15T14:00:00Z",
    createdAt: "2025-01-15T12:00:00Z"
  }
```

### 120 Minutes Later (Automatic Cleanup)

```
Backend:
  CleanupWorker ticker fires (every 1 minute)
    ↓
  Query Redis: GET all "lease:*" keys
    ├─ Found: "lease:abc123" with TTL expired
    ↓
  cleanupContainer("abc123")
    ├─ docker.StopContainer("abc123")
    ├─ docker.RemoveContainer("abc123")
    ├─ containerRepo.Delete("abc123")
    └─ leaseRepo.DeleteLease("lease:abc123")
    ↓
  Log: {
    "container_id": "abc123",
    "action": "cleanup",
    "status": "success",
    "timestamp": "2025-01-15T14:00:05Z"
  }
```

## 🛠️ Development Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Node.js 18+

### Local Setup

```bash
# Navigate to project
cd /Users/aryandhankhar/Documents/dev/containerlease

# Start entire stack
docker-compose up

# Backend should be available at http://localhost:8080
# Frontend should be available at http://localhost:3000
```

### Manual Development

```bash
# Backend
cd backend
REDIS_URL=redis://localhost:6379 go run ./cmd/server

# Frontend (in new terminal)
cd frontend
npm install
npm run dev
```

## 📚 Documentation Files

| File | Content |
|------|---------|
| [README.md](README.md) | Main project overview |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Deep dive into cleanup logic |
| [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) | Directory tree & explanations |
| [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) | Step-by-step implementation roadmap |

## ✅ Implementation Status

### Completed
- ✅ Full directory structure
- ✅ Domain entities & interfaces
- ✅ Clean Architecture foundation
- ✅ Cleanup worker implementation
- ✅ Service layer scaffold
- ✅ Repository layer scaffold
- ✅ Docker Compose setup
- ✅ Frontend structure & hooks
- ✅ Configuration management

### In Progress
- 🟡 Docker client implementation
- 🟡 Repository testing
- 🟡 Handler implementation
- 🟡 Frontend components

### Not Started
- ⭕ Integration tests
- ⭕ Frontend-backend integration
- ⭕ Production deployment
- ⭕ Performance optimization

## 🎓 Learning Resources

### For Understanding This Project

1. **Clean Architecture**: Read the `internal/` layer structure
2. **Background Workers**: See `internal/worker/cleanup_worker.go`
3. **Go Concurrency**: Notice goroutine usage in main.go and worker
4. **Structured Logging**: Check how slog is used throughout
5. **Docker Integration**: Review Docker SDK usage patterns
6. **Redis TTL**: Understand lease expiration mechanism

### Code Entry Points

1. **Backend entry:** `backend/cmd/server/main.go`
2. **Cleanup logic:** `backend/internal/worker/cleanup_worker.go`
3. **Provisioning logic:** `backend/internal/service/container_service.go`
4. **Frontend entry:** `frontend/src/App.tsx`

## 🚨 Important Notes

### Don't Miss
- The cleanup worker runs **regardless of client behavior**
- Redis **TTL is critical** to the cleanup mechanism
- The worker has **retry logic** for failed cleanups
- All errors are **structured-logged** (never silent failures)

### Common Pitfalls
- ❌ Cleanup only on user delete → containers run forever
- ❌ No error handling in Docker operations → silent failures
- ❌ Mixing business logic in handlers → hard to test
- ❌ Ignoring errors → unpredictable behavior

## 📞 Support

For questions about:
- **Architecture:** See [ARCHITECTURE.md](ARCHITECTURE.md)
- **Implementation:** See [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)
- **Structure:** See [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)
- **Getting started:** See [README.md](README.md)

---

**Status:** Ready for implementation. Critical path: Implement Docker client → Test cleanup worker → Build frontend.
