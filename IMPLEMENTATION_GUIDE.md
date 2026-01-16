# ContainerLease - Implementation Guide

## Overview

This document serves as a roadmap for implementing the remaining functionality of ContainerLease following Clean Architecture principles.

## Completed Structure

✅ Full directory structure created  
✅ Domain interfaces defined  
✅ Clean Architecture layers established  
✅ Template implementations provided  
✅ Docker Compose for local development  

## Remaining Implementation Tasks

### Phase 1: Backend Infrastructure (Foundation)

#### 1.1 Complete Docker Client (`internal/infrastructure/docker/client.go`)

**Current Status:** Skeleton with method signatures  
**Tasks:**
- [ ] Implement `CreateContainer(imageType string)`
  - Pull image from Docker registry
  - Create container with resource limits
  - Configure environment variables
  - Start container and return ID
- [ ] Test with real Docker daemon
- [ ] Add error handling for image pull failures
- [ ] Implement container health checks

**Dependencies:** Docker SDK (already in go.mod)

**Example stub to implement:**
```go
func (c *Client) CreateContainer(ctx context.Context, imageType string) (string, error) {
    // 1. Determine image name based on imageType
    // 2. Pull image: c.cli.ImagePull(ctx, imageName, ...)
    // 3. Create container: c.cli.ContainerCreate(ctx, config, hostConfig, ...)
    // 4. Start container: c.cli.ContainerStart(ctx, containerID, ...)
    // 5. Return containerID
}
```

#### 1.2 Complete Repository Layer

**Current Status:** Lease and Container repositories are partially implemented  
**Tasks:**
- [ ] Test `LeaseRepository.CreateLease()` with Redis
- [ ] Test `LeaseRepository.GetExpiredLeases()` (critical for cleanup)
- [ ] Implement proper error handling for Redis operations
- [ ] Add pagination for large number of containers
- [ ] Test TTL expiration behavior

**Key Method - GetExpiredLeases():**
```go
// This is called by CleanupWorker every 1 minute
// Returns container IDs whose leases have expired
func (r *LeaseRepository) GetExpiredLeases() ([]string, error) {
    // Current implementation queries Redis for "lease:*" keys
    // and checks TTL <= 0
}
```

### Phase 2: Handlers & Routes

#### 2.1 Complete Provision Handler

**Current Status:** Basic implementation exists  
**Tasks:**
- [ ] Add request validation (duration limits)
- [ ] Add error responses with proper HTTP status codes
- [ ] Add logging for all operations
- [ ] Test with curl/Postman

#### 2.2 Implement Logs Handler (WebSocket)

**Current Status:** Skeleton only  
**Tasks:**
- [ ] Create WebSocket endpoint for `/ws/logs/{containerId}`
- [ ] Stream Docker container logs in real-time
- [ ] Handle client disconnections gracefully
- [ ] Send logs as JSON: `{timestamp, level, message}`
- [ ] Implement buffer for missed messages

**Implementation pattern:**
```go
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    containerID := r.PathValue("id")  // From route
    
    // 1. Upgrade HTTP to WebSocket
    ws, err := websocket.Upgrader{}.Upgrade(w, r, nil)
    
    // 2. Get Docker logs stream
    logs, err := h.dockerClient.StreamLogs(r.Context(), containerID)
    
    // 3. Read from logs and write to WebSocket
    // 4. Handle connection close
}
```

#### 2.3 Implement Status Handler

**Current Status:** Skeleton only  
**Tasks:**
- [ ] Create GET `/api/containers` endpoint
- [ ] Return list of all active containers
- [ ] Include expiry time and remaining duration
- [ ] Add pagination support

### Phase 3: Cleanup Worker (Most Critical)

#### 3.1 Testing Cleanup Worker

**Current Status:** Implementation complete, needs testing  
**Tasks:**
- [ ] **Unit tests** for `cleanupContainer()` with mock Docker
- [ ] **Integration tests** with real Redis + Docker
- [ ] Test retry logic with Docker API failures
- [ ] Test exponential backoff timing
- [ ] Test structured logging output
- [ ] **Stress test:** Provision 100 containers, verify cleanup

**Test example:**
```go
func TestCleanupWorker_ExpiredLeases(t *testing.T) {
    // 1. Create a lease that expires in 1 second
    // 2. Wait 2 seconds
    // 3. Run cleanup
    // 4. Verify container is deleted
    // 5. Verify lease is removed from Redis
}
```

#### 3.2 Verify Worker Integration

**Current Status:** Started in main.go  
**Tasks:**
- [ ] Start server and verify cleanup worker logs
- [ ] Check Redis TTL behavior
- [ ] Monitor cleanup in action

**Test procedure:**
```bash
# 1. Start server with debug logging
CLEANUP_INTERVAL_MINUTES=1 LOG_LEVEL=debug ./server

# 2. Provision a 2-minute container
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":2}'

# 3. Watch logs for cleanup
# 4. In another terminal, monitor Redis TTL
redis-cli MONITOR
```

### Phase 4: Frontend Integration

#### 4.1 Component Implementation

**Current Status:** Hooks and partial components  
**Tasks:**
- [ ] Implement `ProvisionForm` component
  - Form submission
  - Error handling
  - Loading state
  - Success feedback

- [ ] Implement `ContainerList` component
  - Auto-refresh containers
  - Display expiry countdown
  - Manual delete button

- [ ] Implement `LogViewer` component
  - WebSocket connection
  - Real-time log streaming
  - Scroll to bottom
  - Copy logs functionality

- [ ] Implement `ExpiryTimer` component
  - Countdown display
  - Auto-update
  - Expired state

#### 4.2 Custom Hooks

**Current Status:** Structure only  
**Tasks:**
- [ ] Implement `useContainers()` hook
  - Fetch containers on mount
  - Auto-refresh
  - Expose refetch function

- [ ] Implement `useWebSocket()` hook
  - Connection management
  - Message parsing
  - Reconnection logic

- [ ] Implement `useTimer()` hook
  - Countdown logic
  - Expiry detection

#### 4.3 App Layout

**Tasks:**
- [ ] Create main `App.tsx` component
  - Layout: Provision form + Container list + Selected container logs
  - State management (use custom hooks)
  - Error boundary

### Phase 5: Testing & Validation

#### 5.1 Backend Testing

**Test files to create:**
```
backend/
├── internal/
│   ├── service/
│   │   └── container_service_test.go
│   ├── repository/
│   │   ├── lease_repository_test.go
│   │   └── container_repository_test.go
│   └── worker/
│       └── cleanup_worker_test.go
└── tests/
    ├── integration_test.go
    └── docker/
        └── docker_client_test.go
```

#### 5.2 Frontend Testing

**Test files to create:**
```
frontend/src/
├── components/
│   ├── ProvisionForm.test.tsx
│   ├── ContainerList.test.tsx
│   └── LogViewer.test.tsx
├── hooks/
│   └── useContainers.test.ts
└── services/
    └── containerApi.test.ts
```

### Phase 6: Deployment

#### 6.1 Docker Images

**Tasks:**
- [ ] Verify backend Dockerfile builds
- [ ] Verify frontend Dockerfile builds
- [ ] Test docker-compose.yml integration

#### 6.2 Environment Configuration

**Tasks:**
- [ ] Complete `.env` template
- [ ] Document all environment variables
- [ ] Add validation in config.Load()

## Implementation Priority

### 🔴 Critical Path (Do First)
1. Complete Docker client implementation
2. Test cleanup worker with real containers
3. Implement Logs WebSocket handler
4. Test end-to-end provisioning → cleanup

### 🟡 Important (Do Next)
5. Complete remaining handlers
6. Implement frontend components
7. Add comprehensive tests
8. Test error scenarios

### 🟢 Nice-to-Have
9. Performance optimization
10. Monitoring & metrics
11. Advanced features (pause, extend lease)

## Key File References

### Backend Files to Complete
| File | Purpose | Priority |
|------|---------|----------|
| `internal/infrastructure/docker/client.go` | Docker operations | 🔴 Critical |
| `internal/worker/cleanup_worker.go` | Automatic cleanup | 🔴 Critical |
| `internal/handler/logs.go` | WebSocket logs | 🟡 Important |
| `internal/handler/status.go` | List containers | 🟡 Important |
| `internal/repository/lease_repository.go` | Redis operations | 🔴 Critical |

### Frontend Files to Complete
| File | Purpose | Priority |
|------|---------|----------|
| `src/components/ProvisionForm.tsx` | Container provisioning UI | 🟡 Important |
| `src/components/LogViewer.tsx` | Real-time logs | 🟡 Important |
| `src/hooks/useContainers.ts` | Container state | 🟡 Important |
| `src/App.tsx` | Main app layout | 🟡 Important |

## Code Standards Checklist

### Go Backend
- [ ] No ignored errors (`err` must always be handled)
- [ ] Structured logging with `slog` (never fmt.Println)
- [ ] Proper error wrapping with `fmt.Errorf("%w", err)`
- [ ] Interface-based design (domain layer)
- [ ] Dependency injection in constructors
- [ ] Graceful shutdown handling

### React Frontend
- [ ] Functional components only
- [ ] Strict TypeScript (no `any`)
- [ ] Proper error boundaries
- [ ] Loading states
- [ ] Cleanup in useEffect (return function)

## Quick Command Reference

```bash
# Build backend
cd backend && go build -o bin/server ./cmd/server

# Run backend
go run ./cmd/server

# Run frontend dev
cd frontend && npm run dev

# Run frontend build
npm run build

# Start entire stack
docker-compose up

# View cleanup worker logs
docker logs containerlease-backend-1 | grep cleanup

# Test provision endpoint
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":5}'

# Redis inspect
redis-cli KEYS "lease:*"
redis-cli TTL "lease:abc123"
```

## Debugging Tips

### Cleanup Not Running?
1. Check logs for worker startup message
2. Verify Redis connection
3. Check cleanup interval configuration
4. Ensure TTL is set properly in Redis

### Docker Container Not Cleaning Up?
1. Verify Docker socket access
2. Check Docker API errors in logs
3. Manually verify with `docker ps`
4. Check container creation logic

### WebSocket Logs Not Streaming?
1. Verify WebSocket connection in browser DevTools
2. Check container ID is valid
3. Verify Docker logs available
4. Check for connection errors in logs

---

**Next Step:** Start with Phase 1 (Docker client implementation), then move to testing the cleanup worker. That's the critical path.
