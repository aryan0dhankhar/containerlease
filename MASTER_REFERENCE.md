# ContainerLease - Master Reference Document

## 🎯 Project Overview

**ContainerLease** is a web platform where developers can click a button to provision temporary Docker containers (Ubuntu/Alpine) for fixed durations (e.g., 2 hours). After expiry, a **background worker automatically kills and removes the container** to free resources.

**Stack:** Go 1.21+ backend, React + TypeScript frontend, Redis cache, Docker integration, WebSockets for logs.

---

## 📍 The Most Important File

### ⭐⭐⭐ `backend/internal/worker/cleanup_worker.go`

This is the **HEART** of ContainerLease. It:
- Runs every 1 minute in background
- Queries Redis for expired container leases
- Automatically stops and removes expired containers
- Implements retry logic with exponential backoff
- Ensures NO containers run indefinitely

**Why it's separate:** Cleanup happens automatically regardless of user action. Without this, containers would run forever if users don't manually delete them.

---

## 📂 Complete File Listing

### 📄 Documentation (8 files)
```
./README.md                    # Features, tech stack, API, quick start
./SUMMARY.md                   # Executive summary & status
./ARCHITECTURE.md              # Cleanup logic deep dive ⭐⭐⭐
./PROJECT_STRUCTURE.md         # Directory tree & explanations
./IMPLEMENTATION_GUIDE.md      # Step-by-step implementation roadmap
./DIAGRAMS.md                  # Visual diagrams
./INDEX.md                     # Documentation index & navigation
./COMPLETION_REPORT.md         # Project completion summary
```

### 🔧 Backend (19 files)

**Entry Point:**
```
./backend/cmd/server/main.go
```

**Domain Layer (entities & contracts):**
```
./backend/internal/domain/container.go
```

**Handler Layer (HTTP requests):**
```
./backend/internal/handler/provision.go
```

**Service Layer (business logic):**
```
./backend/internal/service/container_service.go
```

**Repository Layer (data access):**
```
./backend/internal/repository/lease_repository.go
./backend/internal/repository/container_repository.go
```

**Infrastructure Layer (external clients):**
```
./backend/internal/infrastructure/docker/client.go
./backend/internal/infrastructure/redis/client.go
./backend/internal/infrastructure/logger/logger.go
```

**Worker Layer (background jobs):**
```
./backend/internal/worker/cleanup_worker.go  ⭐⭐⭐
```

**Configuration & Dependencies:**
```
./backend/pkg/config/config.go
./backend/go.mod
```

**Build & Deployment:**
```
./backend/Dockerfile
./backend/config/.env.example
```

### ⚛️ Frontend (7 files)

**Components:**
```
./frontend/src/components/ProvisionForm.tsx
./frontend/src/components/ContainerList.tsx
```

**Services (API integration):**
```
./frontend/src/services/containerApi.ts
```

**Types:**
```
./frontend/src/types/container.ts
```

**Configuration:**
```
./frontend/vite.config.ts
./frontend/tsconfig.json
./frontend/package.json
```

### 🐳 Infrastructure (3 files)
```
./docker-compose.yml           # Local dev: Redis + Backend + Frontend
./frontend/Dockerfile          # Frontend container
./backend/Dockerfile           # Backend container
```

### ⚙️ Root Configuration
```
./.gitignore
```

---

## 🏗️ Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND (React + TypeScript)                               │
│ - Provision Form | Container List | Log Viewer              │
└────────────────────────┬────────────────────────────────────┘
                         │
          REST API + WebSocket
                         │
┌────────────────────────▼────────────────────────────────────┐
│ BACKEND (Go 1.21+)                                          │
│                                                             │
│ Handler Layer (HTTP)                                        │
│  ├─ POST /api/provision  ├─ GET /api/containers            │
│  └─ WS /ws/logs/{id}                                        │
│         │                                                   │
│ Service Layer (Business Logic)                              │
│  ├─ ContainerService (provisioning)                         │
│  └─ LifecycleService (management)                           │
│         │                                                   │
│ Repository Layer (Data Access)                              │
│  ├─ LeaseRepository     ├─ ContainerRepository              │
│         │                                                   │
│ Infrastructure Layer (External Clients)                     │
│  ├─ DockerClient        ├─ RedisClient  ├─ Logger          │
│         │                                                   │
│ ⭐ Worker Layer (Background Jobs)                           │
│  └─ CleanupWorker (runs every 1 min) ⭐⭐⭐                  │
│         │                                                   │
└────────────────────────┬────────────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
       Docker          Redis         (Docker)
       Daemon          Cache          Logs
```

---

## 🔄 How It Works

### 1. User Provisions Container
```
Frontend: Click "Provision"
  ↓
POST /api/provision {imageType: "ubuntu", durationMinutes: 120}
  ↓
Backend: ProvisionHandler
  ↓
ContainerService.ProvisionContainer()
  ├─ Docker: Create container
  ├─ Repository: Save container
  └─ Redis: Store lease with TTL=7200s
  ↓
Response: {id: "abc123", expiryTime: "2025-01-15T14:00:00Z"}
```

### 2. User Views Logs
```
Frontend: Select container → View logs
  ↓
WS /ws/logs/abc123
  ↓
Backend: LogsHandler (WebSocket)
  ├─ Connect to Docker logs
  └─ Stream to client real-time
```

### 3. Container Expires (2 hours later)
```
Backend: CleanupWorker ticker fires (every 1 min)
  ↓
Query Redis: GET all "lease:*" keys with TTL=0
  ↓
Found: lease:abc123 (EXPIRED)
  ↓
cleanupContainer("abc123")
  ├─ docker.StopContainer()
  ├─ docker.RemoveContainer()
  ├─ containerRepo.Delete()
  └─ leaseRepo.DeleteLease()
  ↓
Log: {container_id: "abc123", status: "success"}
```

---

## 📚 Reading Guide

### For 5-Minute Overview
1. [SUMMARY.md](SUMMARY.md) - Quick summary
2. This document - Master reference

### For Understanding Architecture
1. [ARCHITECTURE.md](ARCHITECTURE.md) - Cleanup logic deep dive
2. [DIAGRAMS.md](DIAGRAMS.md) - Visual system diagrams
3. [backend/internal/worker/cleanup_worker.go](backend/internal/worker/cleanup_worker.go) - Read the code

### For Building the Project
1. [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) - Phase-by-phase roadmap
2. [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) - Directory explanations
3. Start with Phase 1: Implement Docker client

### For Code Navigation
1. [INDEX.md](INDEX.md) - Documentation index
2. [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) - File organization

---

## 🔴 Critical Components (Must Implement First)

### 1. Docker Client (`backend/internal/infrastructure/docker/client.go`)
- Container creation with resource limits
- Container stopping and removal
- Log streaming

### 2. Cleanup Worker (`backend/internal/worker/cleanup_worker.go`)
- Already implemented ✅
- Needs thorough testing
- Monitor expiry detection from Redis

### 3. Repository Layer (`backend/internal/repository/*.go`)
- Redis lease storage with TTL
- Expired lease query (GetExpiredLeases)
- Critical for cleanup detection

### 4. Handler Layer (`backend/internal/handler/*.go`)
- Provision endpoint
- Logs WebSocket endpoint
- Status endpoint

### 5. Frontend Components (`frontend/src/components/`)
- Provision form
- Container list
- Log viewer

---

## ⚙️ Configuration

### Environment Variables
Located in `backend/config/.env.example`:
```
SERVER_PORT=8080
REDIS_URL=redis://localhost:6379
DOCKER_HOST=unix:///var/run/docker.sock
CLEANUP_INTERVAL_MINUTES=1
LOG_LEVEL=info
```

### Local Development
```bash
docker-compose up
```

Starts:
- Redis on port 6379
- Backend on port 8080
- Frontend on port 3000

---

## 🎓 Key Architectural Decisions

### Why Clean Architecture?
- **Separation of Concerns:** Each layer has one job
- **Testability:** Easy to mock dependencies
- **Maintainability:** Code organized logically
- **Scalability:** Add features without major refactoring

### Why Background Worker?
- **Automatic Cleanup:** No dependency on user action
- **Time-Guaranteed:** Cleanup within set interval
- **Resilient:** Retry logic handles failures
- **Observable:** All operations logged

### Why Redis TTL?
- **Automatic Expiration:** No external cron jobs needed
- **Distributed:** Works across multiple instances
- **Simple:** Straightforward TTL management
- **Queryable:** Easy to find expired leases

### Why WebSocket for Logs?
- **Real-time:** Immediate log display
- **Efficient:** Push vs polling
- **Persistent:** Connection stays open
- **Streaming:** Full log history available

---

## 📊 Project Status

### ✅ Completed
- Full directory structure
- All domain entities defined
- Clean Architecture foundation
- Cleanup worker implementation
- Service layer scaffold
- Repository layer scaffold
- Handler skeleton
- Frontend structure
- Docker Compose setup
- Complete documentation (8 files)
- Implementation roadmap

### 🔄 In Progress
- Docker client implementation
- Repository testing
- Handler implementation
- Frontend components

### ⏳ Not Started
- Integration tests
- Frontend-backend integration
- Production deployment
- Performance optimization

---

## 🚀 Next Steps (In Order)

1. **Phase 1:** Implement Docker client
   - Study Docker SDK
   - Implement CreateContainer, StopContainer, RemoveContainer
   - Test with real Docker

2. **Phase 2:** Complete repositories
   - Test LeaseRepository with Redis
   - Verify TTL behavior
   - Test GetExpiredLeases()

3. **Phase 3:** Test cleanup worker
   - Unit tests with mocks
   - Integration tests with real services
   - End-to-end provisioning → cleanup

4. **Phase 4:** Build frontend
   - Implement components
   - Connect to API
   - Test WebSocket logs

5. **Phase 5:** Testing & validation
   - Comprehensive test coverage
   - Error scenario testing
   - Performance testing

6. **Phase 6:** Deployment
   - Docker image verification
   - Kubernetes configs
   - Production setup

---

## 📞 Quick Links

### Documentation Files
- [README.md](README.md) - Main documentation
- [SUMMARY.md](SUMMARY.md) - Quick overview
- [ARCHITECTURE.md](ARCHITECTURE.md) - Cleanup logic ⭐⭐⭐
- [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) - Build guide
- [DIAGRAMS.md](DIAGRAMS.md) - Visual diagrams
- [INDEX.md](INDEX.md) - Navigation
- [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) - File organization

### Critical Files
- [backend/internal/worker/cleanup_worker.go](backend/internal/worker/cleanup_worker.go) - Cleanup logic ⭐⭐⭐
- [backend/internal/service/container_service.go](backend/internal/service/container_service.go) - Provisioning
- [backend/cmd/server/main.go](backend/cmd/server/main.go) - Entry point
- [backend/internal/repository/lease_repository.go](backend/internal/repository/lease_repository.go) - Redis integration

---

## 💡 Remember

### The Cleanup Worker is Key
The success of ContainerLease depends on the cleanup worker:
- Runs automatically every 1 minute
- Queries Redis for expired leases
- Stops and removes containers
- Ensures resources are freed
- No manual intervention required

### Test Thoroughly
Before moving to next phase:
- Unit test each component
- Integration test with real services
- Test error scenarios
- Monitor logs closely

### Follow Clean Architecture
Maintain layer separation:
- Handlers only handle HTTP
- Services only handle business logic
- Repositories only handle data access
- Infrastructure only wraps external clients

---

## 🎉 You're Ready to Build!

All scaffolding is complete. The project is documented and structured. Start with [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) Phase 1.

**Current Location:** `/Users/aryandhankhar/Documents/dev/containerlease/`

Good luck! 🚀

---

**Last Updated:** January 15, 2026
**Status:** Ready for Implementation
**Next:** Phase 1 - Implement Docker Client
