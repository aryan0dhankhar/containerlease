# Architecture Deep Dive

## Cleanup Logic - The Heart of ContainerLease

### Location: `backend/internal/worker/cleanup_worker.go`

The cleanup logic is **not** in the HTTP handlers or service layer. Instead, it runs as an **independent background worker** that:

1. **Periodically runs** on a Go Ticker (every 1 minute by default)
2. **Queries Redis** for expired container leases
3. **Orchestrates cleanup** of Docker containers
4. **Implements retry logic** with exponential backoff
5. **Logs all operations** with structured context

### Execution Flow

```
┌─────────────────────────────────────────────────────────┐
│ main.go initializes CleanupWorker and starts it         │
│ in a background goroutine                               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ CleanupWorker.Start()      │
        │ (runs forever in goroutine)│
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ Timer fires every 1 minute │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ cleanupExpiredContainers()         │
        │  - Query Redis for expired leases │
        │  - Get container IDs               │
        └────────────┬─────────────────────┘
                     │
                     ▼
     ┌──────────────────────────────────────────┐
     │ For each expired container:              │
     │  cleanupContainer(containerID)           │
     │  ├─ Retry loop (up to 3 times)          │
     │  └─ Exponential backoff on failure      │
     └──────────────┬───────────────────────────┘
                    │
                    ▼
       ┌────────────────────────────────┐
       │ performCleanup():               │
       │  1. docker.StopContainer()      │
       │  2. docker.RemoveContainer()    │
       │  3. containerRepo.Delete()      │
       │  4. leaseRepo.DeleteLease()     │
       │  5. Log result                  │
       └────────────────────────────────┘
```

### Why a Separate Worker?

**Problem with service-based cleanup:**
```go
// ❌ BAD: Cleanup only happens when someone calls the service
func (s *ContainerService) ProvisionsContainer(...) {
    // ... provision container ...
    // Cleanup only runs if we manually call:
    // s.DeleteContainer(id) 
    // But what if nobody calls this? Container runs forever!
}
```

**Solution with background worker:**
```go
// ✅ GOOD: Cleanup runs automatically every 1 minute regardless
go cleanupWorker.Start(ctx)  // Starts in main.go

// Every 1 minute, it checks Redis for expired leases
// and automatically cleans them up
```

### Code Walkthrough

```go
// 1. Worker starts and loops forever
func (w *CleanupWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)  // 1 minute
    
    for {
        select {
        case <-ctx.Done():
            return  // Graceful shutdown
        case <-ticker.C:
            w.cleanupExpiredContainers(ctx)  // Check every tick
        }
    }
}

// 2. Query Redis for expired leases
func (w *CleanupWorker) cleanupExpiredContainers(ctx context.Context) {
    expiredLeases, err := w.leaseRepository.GetExpiredLeases()
    // Returns: ["container-id-1", "container-id-2", ...]
}

// 3. Clean up each one with retries
func (w *CleanupWorker) cleanupContainer(ctx context.Context, containerID string) {
    for attempt := 1; attempt <= w.maxRetries; attempt++ {
        if w.performCleanup(ctx, containerID) {
            return  // Success!
        }
        
        // Exponential backoff: 1s, 4s, 9s
        backoff := time.Duration(attempt*attempt) * time.Second
        time.Sleep(backoff)
    }
}

// 4. Actual cleanup steps
func (w *CleanupWorker) performCleanup(ctx context.Context, containerID string) bool {
    // Step 1: Stop the Docker container
    if err := w.dockerClient.StopContainer(containerID); err != nil {
        return false
    }
    
    // Step 2: Remove the Docker container
    if err := w.dockerClient.RemoveContainer(containerID); err != nil {
        return false
    }
    
    // Step 3: Remove from container repository
    if err := w.containerRepository.Delete(containerID); err != nil {
        return false
    }
    
    // Step 4: Remove lease from Redis
    if err := w.leaseRepository.DeleteLease(fmt.Sprintf("lease:%s", containerID)); err != nil {
        return false
    }
    
    return true  // Success!
}
```

## Integration with Main Application

### Initialization (`main.go`)

```go
func main() {
    // 1. Initialize dependencies
    dockerClient := docker.NewClient(cfg.DockerHost)
    leaseRepo := repository.NewLeaseRepository(redisClient)
    containerRepo := repository.NewContainerRepository(redisClient)
    
    // 2. Create cleanup worker
    cleanupWorker := worker.NewCleanupWorker(
        leaseRepo,
        containerRepo,
        dockerClient,
        log,
        time.Duration(cfg.CleanupIntervalMinutes) * time.Minute,
    )
    
    // 3. Start it in background!
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go cleanupWorker.Start(ctx)  // Runs in parallel with HTTP server
    
    // 4. Start HTTP server (runs on main thread)
    server.ListenAndServe()
}
```

## Data Flow: Lease Expiry

### Timeline Example

**T=0 (User clicks "Provision")**
```
POST /api/provision { imageType: "ubuntu", durationMinutes: 120 }
    ↓
ContainerService.ProvisionContainer()
    ├─ Create Docker container → "abc123"
    ├─ Save to repository
    └─ Create Redis lease: "lease:abc123" with TTL=7200s (2 hours)
    ↓
Response: { id: "abc123", expiryTime: "2025-01-15T14:00:00Z" }
```

**T=120 minutes (2 hours later)**
```
CleanupWorker ticker fires
    ↓
Query Redis: GET all "lease:*" keys
    ├─ Found: "lease:abc123" with TTL=0 (EXPIRED!)
    ↓
cleanupContainer("abc123")
    ├─ docker.StopContainer("abc123")        ← Kill process
    ├─ docker.RemoveContainer("abc123")      ← Clean filesystem
    ├─ containerRepo.Delete("abc123")        ← Remove from repo
    └─ leaseRepo.DeleteLease("lease:abc123") ← Remove from Redis
    ↓
Log: { container_id: "abc123", action: "cleanup", status: "success" }
```

## Redis Schema

```
Key: "lease:{containerID}"
Type: String + TTL
Value: { containerID, expiryTime, imageType, durationMinutes }
TTL: Set to match DurationMinutes

Example:
SET lease:abc123 '{"containerID":"abc123","expiryTime":"2025-01-15T14:00:00Z",...}' EX 7200

# After 2 hours:
# Redis automatically expires this key
# GetExpiredLeases() queries for keys with TTL < now()
```

## Error Handling Strategy

### Cleanup Failure Scenarios

| Scenario | Action |
|----------|--------|
| Docker API unreachable | Retry with exponential backoff |
| Container already stopped | Continue (idempotent) |
| Redis delete fails | Log error, mark for manual cleanup |
| Partial cleanup (e.g., stopped but not removed) | Retry entire cleanup |

### Structured Logging

```json
{
  "timestamp": "2025-01-15T12:00:00Z",
  "level": "info",
  "container_id": "abc123",
  "action": "cleanup",
  "status": "success",
  "attempt": 1,
  "duration_ms": 234
}
```

## Scalability Considerations

### Single Machine
- Works perfectly with `CLEANUP_INTERVAL_MINUTES=1`
- No race conditions (single process)

### Multiple Instances
If scaling to multiple backend servers:
1. **Use Redis distributed lock** for cleanup coordination
2. **Only one instance runs cleanup** at a time
3. Prevents duplicate deletions

```go
// Future enhancement
func (w *CleanupWorker) acquireCleanupLock(ctx context.Context) bool {
    // SET lock "cleanup-lock" NX EX 90
    // Only proceed if lock acquired
}
```

---

**Key Takeaway:** The cleanup worker is the **automatic garbage collector** of ContainerLease. It ensures no containers run indefinitely, regardless of client behavior.
