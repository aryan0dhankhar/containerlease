# ContainerLease - Temporary Docker Container Provisioning Platform

A modern web platform where developers can provision temporary Docker containers with strict resource limits, automatic lifecycle management, and real-time log streaming. Built with Go backend, React frontend, and Redis state management.

## ✨ Features

### Core Capabilities
- **🐳 Docker Provisioning**: Native Docker SDK integration for container management
- **⏱️ Time-Limited Leases**: Automatic expiration with configurable TTLs (5-120 minutes)
- **📊 Resource Quotas**: Hard CPU/memory limits enforced per container
- **💰 Usage-Based Billing**: Cost calculated on actual runtime at termination
- **🧹 Garbage Collection**: Background reconciliation removes expired/orphaned containers
- **📡 Real-Time Logs**: WebSocket streaming with heartbeat keepalive
- **🔒 Security**: Image allowlist, CORS enforcement, origin validation
- **📈 Observability**: Structured JSON logging with request IDs

### Resource Management
- **CPU**: 250m - 2000m millicores (0.25 - 2 CPU cores)
- **Memory**: 256MB - 2GB
- **Duration**: 5 - 120 minutes (configurable)
- **Allowed Images**: Ubuntu 22.04, Alpine Linux (extensible)

### Frontend Features
- Live dashboard with countdown timers per container
- Real-time log viewer with connection state indicators
- Resource selector with validation
- Cost estimation vs. final billing display
- Responsive design with mobile support

## Architecture Overview

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────┐
│              HTTP / WebSocket Layer                 │
│        (Handlers, Middleware, Route Handlers)       │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────┐
│           Business Logic Layer (Service)            │
│  (Provisioning, Cleanup, Lifecycle Management)      │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────┐
│         Data Access Layer (Repository)              │
│    (Redis, Docker SDK abstraction)                  │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────┐
│         Infrastructure Layer                        │
│  (Docker Client, Redis Client, Logger)              │
└─────────────────────────────────────────────────────┘
```

## Project Structure

### Backend (Go 1.21+)

```
backend/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go
├── internal/                # Private application code
│   ├── domain/              # Domain entities and interfaces
│   │   ├── container.go     # Container domain model
│   │   └── lease.go         # Lease domain model
│   ├── handler/             # HTTP handlers
│   │   ├── provision.go     # POST /provision handler
│   │   ├── logs.go          # WebSocket handler
│   │   └── status.go        # GET /containers handler
│   ├── service/             # Business logic
│   │   ├── container_service.go  # Core provisioning logic
│   │   └── lifecycle_service.go  # Cleanup & expiry logic
│   ├── repository/          # Data access layer
│   │   ├── lease_repository.go   # Redis-backed lease storage
│   │   └── container_repository.go
│   ├── infrastructure/      # External integrations
│   │   ├── docker/          # Docker SDK wrapper
│   │   │   ├── client.go
│   │   │   └── container.go
│   │   ├── redis/           # Redis client wrapper
│   │   │   └── client.go
│   │   ├── logger/          # Structured logging
│   │   │   └── logger.go
│   ├── middleware/          # HTTP middleware
│   │   ├── error_handler.go
│   │   └── request_logger.go
│   └── worker/              # Background jobs
│       └── cleanup_worker.go    # ⭐ CLEANUP LOGIC RESIDES HERE
├── pkg/                     # Reusable packages
│   ├── errs/                # Custom error types
│   │   └── errors.go
│   ├── config/              # Configuration management
│   │   └── config.go
│   └── dto/                 # Data transfer objects
│       └── container.go
├── config/
│   ├── .env.example
│   └── config.yaml
├── go.mod
├── go.sum
└── Dockerfile
```

### Frontend (React + TypeScript + Vite)

```
frontend/
├── src/
│   ├── components/          # React components
│   │   ├── ProvisionForm.tsx     # Container provisioning UI
│   │   ├── ContainerList.tsx     # Active containers display
│   │   ├── LogViewer.tsx         # Real-time log streaming
│   │   └── ExpiryTimer.tsx       # Countdown timer
│   ├── hooks/               # Custom React hooks
│   │   ├── useContainers.ts      # Container state management
│   │   ├── useWebSocket.ts       # WebSocket management
│   │   └── useTimer.ts           # Timer management
│   ├── services/            # API clients
│   │   ├── containerApi.ts       # REST API client
│   │   └── logService.ts         # WebSocket log client
│   ├── types/               # TypeScript interfaces
│   │   ├── container.ts
│   │   └── api.ts
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── public/
├── vite.config.ts
├── tsconfig.json
├── package.json
└── Dockerfile
```

## Core Components Explanation

### 1. **Docker Cleanup Logic** ⭐

**Location:** `backend/internal/worker/cleanup_worker.go`

The cleanup logic is a background **Worker** that:
- Runs on a **Go Ticker** (checks every 1 minute)
- Queries Redis for expired container leases
- Calls Docker API to kill and remove containers
- Handles retry logic and error recovery
- Logs all operations using structured logging

**Pseudo-code flow:**
```
Every 1 minute:
  1. Query Redis: GET all keys with prefix "lease:*"
  2. For each lease, check if TTL expired or `expiry_time < now()`
  3. If expired:
     a. Fetch container ID from Redis
     b. Call docker.StopContainer(containerID)
     c. Call docker.RemoveContainer(containerID)
     d. Delete lease from Redis
     e. Log success with context (container ID, duration, cleanup time)
  4. If error during cleanup, retry logic with exponential backoff
```

### 2. **Clean Architecture Principles**

#### **Handler Layer** (`internal/handler/`)
- Receives HTTP requests/WebSocket connections
- Validates input (request unmarshalling)
- Calls service layer
- Returns HTTP responses

#### **Service Layer** (`internal/service/`)
- **ContainerService**: Orchestrates container provisioning
  - Calls Docker infrastructure to create container
  - Stores lease in Redis repository
  - Returns container ID & expiry time
- **LifecycleService**: Manages container lifetime
  - Monitors lease expiry
  - Triggers cleanup (coordinated with Worker)

#### **Repository Layer** (`internal/repository/`)
- Abstracts data persistence
- Redis operations for lease storage/retrieval
- Lease key format: `lease:{containerId}` with TTL

#### **Infrastructure Layer** (`internal/infrastructure/`)
- **Docker Client**: Wraps Docker SDK
  - Container creation with resource limits
  - Log streaming abstraction
  - Graceful shutdown
- **Redis Client**: Connection pooling, error handling
- **Logger**: Structured logging with context

### 3. **Data Flow**

```
Frontend Click "Provision" Button
    ↓
REST API: POST /provision {imageType: "ubuntu", duration: 120}
    ↓
ProvisionHandler.Handle()
    ↓
ContainerService.Provision()
    ├─ Docker: Create container
    ├─ Redis: Store lease with TTL
    └─ Return {id, expiryTime}
    ↓
Response to Frontend
    ↓
Frontend WebSocket: GET /logs/{containerId}
    ↓
LogsHandler (WebSocket)
    ├─ Stream container stdout
    └─ Broadcast to client in real-time
    ↓
[Background] CleanupWorker.Run() (every 1 minute)
    ├─ Detect expired leases in Redis
    ├─ Stop & remove Docker containers
    └─ Clean up Redis entries
```

## Tech Stack Details

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Backend** | Go 1.21+ | Goroutines for concurrency, native Docker SDK |
| **Frontend** | React + TypeScript | Functional components, strict typing |
| **State** | Redis | Session storage, expiry TTL, lease tracking |
| **Container Runtime** | Docker SDK (Go) | Container provisioning & lifecycle |
| **Communication** | REST + WebSocket (Gorilla) | Control API + real-time logs |
| **Logging** | slog (stdlib) | Structured JSON logging |

## API Endpoints

### REST Endpoints

```
POST /api/provision
  Request: { imageType: string, durationMinutes: number }
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

## Error Handling Strategy

- **Domain Errors**: Custom types in `pkg/errs/` (e.g., `ContainerNotFound`, `DockerError`)
- **Service Layer**: Propagates errors with context
- **Handler Layer**: Converts errors to HTTP status codes
- **Middleware**: Catches panics, logs with structured context

## Deployment & Running

```bash
# Backend
cd backend
go build -o bin/server ./cmd/server
./bin/server

# Frontend
cd frontend
npm run dev    # Development
npm run build  # Production build
```

## Environment Variables

See [config/.env.example](config/.env.example) for required variables:
- `REDIS_URL`
- `DOCKER_HOST`
- `CLEANUP_INTERVAL_MINUTES`
- `SERVER_PORT`

---

**Next Steps:** Implement domain entities, service interfaces, and the cleanup worker with Redis integration.
