# ContainerLease - Visual Architecture Diagrams

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         FRONTEND (React + TypeScript)               │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ ProvisionForm    │  │ ContainerList    │  │   LogViewer      │ │
│  │  - Input form    │  │  - Active list   │  │ - WebSocket logs │ │
│  │  - Submit        │  │  - Delete btn    │  │ - Auto-scroll    │ │
│  └────────┬─────────┘  └─────────┬────────┘  └────────┬─────────┘ │
│           │                      │                    │            │
│           └──────────────────────┼────────────────────┘            │
│                                  │                                 │
│                    ┌─────────────▼──────────────┐                 │
│                    │   containerApi Service    │                 │
│                    │  - REST calls              │                 │
│                    │  - WebSocket mgmt          │                 │
│                    └─────────────┬──────────────┘                 │
│                                  │                                 │
└──────────────────────────────────┼─────────────────────────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │                            │
      ┌─────────────┴────────┐                 │
      │                      │                 │
  REST API              WebSocket          (Localhost)
 (JSON)              (Real-time logs)
      │                      │                 │
      │                      │                 │
┌─────▼──────────────────────▼─────────────────▼─────────────────┐
│                      BACKEND (Go 1.21+)                         │
│                                                                 │
│  ┌─────────────────────── HTTP Handlers ──────────────────┐    │
│  │  POST /provision  GET /containers  DELETE /containers  │    │
│  │  WS /logs/{id}                                         │    │
│  └──────────────────────────┬──────────────────────────────┘    │
│                             │                                   │
│  ┌──────────────────────── Services ─────────────────────┐      │
│  │  ContainerService          LifecycleService          │      │
│  │  - Provision orchestration  - Cleanup triggers       │      │
│  │  - Lease creation           - Expiry checks          │      │
│  └───────────────┬─────────────────────────┬────────────┘      │
│                  │                         │                   │
│  ┌──────────────▼─────────────────────────▼──────────────┐     │
│  │  Repositories (Data Access Layer)                    │     │
│  │  ┌──────────────────┐  ┌──────────────────────────┐  │     │
│  │  │ LeaseRepository  │  │ ContainerRepository      │  │     │
│  │  │ - CreateLease()  │  │ - Save()                 │  │     │
│  │  │ - GetExpiredLeases() - GetByID()              │  │     │
│  │  │ - DeleteLease()  │  │ - Delete()               │  │     │
│  │  └──────────────────┘  └──────────────────────────┘  │     │
│  └──────────────┬──────────────────────────┬────────────┘      │
│                 │                          │                   │
│  ┌──────────────▼──────────────────────────▼──────────────┐    │
│  │  Infrastructure (External Clients)                    │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │    │
│  │  │ DockerClient │  │ RedisClient  │  │   Logger   │  │    │
│  │  │ - Create     │  │ - Set/Get    │  │ - Structured│  │    │
│  │  │ - Stop       │  │ - Keys       │  │   JSON     │  │    │
│  │  │ - Remove     │  │ - TTL        │  │   logging  │  │    │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │    │
│  └──────────┬──────────────────────────┬────────────────┘     │
│             │                          │                      │
│  ┌──────────▼─────────────────────────▼──────────────┐        │
│  │  ⭐⭐⭐ BACKGROUND WORKER (Special Layer)  │        │
│  │  CleanupWorker                                   │        │
│  │  - Runs every 1 minute                           │        │
│  │  - Queries Redis for expired leases              │        │
│  │  - Triggers cleanup for each expired container   │        │
│  │  - Retry logic + exponential backoff             │        │
│  └──────────┬──────────────────────────┬────────────┘        │
│             │                          │                      │
└─────────────┼──────────────────────────┼──────────────────────┘
              │                          │
      ┌───────▼───────┐          ┌──────▼────────┐
      │   Docker      │          │     Redis     │
      │   Daemon      │          │   Cache       │
      │               │          │               │
      │  Containers   │          │  Leases:      │
      │  Running      │          │  - TTL=120min │
      │  Volumes      │          │  - Auto exp.  │
      │               │          │               │
      └───────────────┘          └───────────────┘
```

## Cleanup Worker Sequence Diagram

```
Time: Every 1 minute

┌─────────────────┐                                    ┌─────────────────┐
│  CleanupWorker  │                                    │   Redis & Docker│
└────────┬────────┘                                    └────────┬────────┘
         │                                                      │
         │─────── Timer fires (every 1 minute) ──────────────────────►
         │
         │─────── cleanupExpiredContainers() ─────────────────────────►
         │                                      │
         │◄───── GET lease:* (all lease keys) ◄─┤
         │       Returns: ["lease:abc", "lease:def"]
         │       (filters by TTL < 0)
         │
         │─────── For each expired lease ─────────────────────────────►
         │        cleanupContainer("abc")
         │
         │        Retry Loop (max 3 times)
         │        ┌─────────────────────────┐
         │        │ Attempt 1:              │
         │        │                         │
         │        ├─ docker.StopContainer() ──────────────────────────►
         │        │  (stop process)         │
         │        │
         │        ├─ docker.RemoveContainer() ───────────────────────►
         │        │  (remove filesystem)    │
         │        │
         │        ├─ containerRepo.Delete() ──────────────────────────►
         │        │  (clean repository)     │
         │        │
         │        ├─ leaseRepo.DeleteLease() ────────────────────────►
         │        │  (remove from Redis)    │
         │        │                         │
         │        │ On success: Break       │
         │        │ On failure: Retry       │
         │        └─────────────────────────┘
         │
         │─────── All cleanup done, log results ──────────────────────►
         │        { container_id, action, status, timestamp }
         │
         │ Wait 59 seconds
         │
         │─────── Timer fires again ─────────────────────────────────►
         │
```

## Data Flow: Container Lifetime

```
T=0min                     T=60min                   T=120min (EXPIRED)
User Provisions            Running                   Cleanup Worker
   │                          │                            │
   ▼                          ▼                            ▼
┌─────────────────────┐  ┌──────────────┐         ┌────────────────────┐
│ Frontend: Click      │  │ Container:   │         │ Worker checks Redis│
│ "Provision"         │  │ - Running    │         │ for TTL=0          │
└──────────┬──────────┘  │ - Streaming  │         └─────────┬──────────┘
           │             │   logs       │                    │
           │             └──────┬───────┘                    │
      REST API                  │                            │
POST /provision            WebSocket: logs                   │
  {                         streaming                        │
   imageType: "ubuntu"          │                            │
   duration: 120             User viewing                   ▼
  }                         real-time logs          ┌────────────────────┐
           │                                        │ Stop Container     │
           ▼                                        │ Remove Container   │
┌──────────────────────────────┐                   │ Delete Lease       │
│ ContainerService:            │                   │ Delete from Repo   │
│ 1. docker.CreateContainer()  │                   └────────┬───────────┘
│ 2. container.Save()          │                           │
│ 3. lease.CreateLease()       │                           ▼
└────────────┬─────────────────┘                   ┌────────────────────┐
             │                                     │ Log Cleanup        │
             ▼                                     │ {                  │
    ┌─────────────────────┐                       │  container_id: abc │
    │ Redis:              │                       │  status: "success" │
    │ Set lease:abc123 {} │                       │  timestamp: T=120  │
    │ EX 7200s (120min)   │                       │ }                  │
    │                     │                       └────────────────────┘
    │ TTL countdown...    │
    │ 7200 → 6000 → ... → 0
    └─────────────────────┘
```

## Clean Architecture Dependency Flow

```
┌─────────────────────────────────────────────────────────────┐
│ HTTP Request/Response                                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │    HANDLER LAYER     │ (Receives & validates)
          │ ProvisionHandler     │
          │ - Parse request      │
          │ - Call service       │
          │ - Return response    │
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │   SERVICE LAYER      │ (Business logic)
          │ ContainerService     │
          │ - Orchestrate        │
          │ - Coordinate         │
          └──────────┬───────────┘
                     │
      ┌──────────────┼──────────────┐
      │              │              │
      ▼              ▼              ▼
   ┌──────────┐  ┌──────────┐  ┌──────────────┐
   │Repository│  │Repository│  │  DOMAIN     │
   │ LAYER    │  │  LAYER   │  │  ENTITIES   │
   │ LAYER    │  │  LAYER   │  │             │
   │ Lease    │  │Container │  │ Container   │
   │ Repo     │  │ Repo     │  │ Lease       │
   └────┬─────┘  └────┬─────┘  └──────┬──────┘
        │             │               │
        │             │               │
        ▼             ▼               ▼
   ┌──────────────────────────────────────────┐
   │   INFRASTRUCTURE LAYER                   │
   │ ┌──────────────┐  ┌──────────────────┐  │
   │ │ RedisClient  │  │  DockerClient    │  │
   │ │ - Set/Get    │  │  - Create        │  │
   │ │ - Delete     │  │  - Stop          │  │
   │ │ - Keys       │  │  - Remove        │  │
   │ └──────────────┘  └──────────────────┘  │
   └──────────────────────────────────────────┘
            │                    │
            ▼                    ▼
       ┌─────────┐          ┌────────┐
       │ Redis   │          │ Docker │
       │ Process │          │ Daemon │
       └─────────┘          └────────┘


KEY PRINCIPLE: 
Dependencies point inward, not outward.
- Handlers depend on Services
- Services depend on Repositories & Domain
- Repositories depend on Infrastructure
- Infrastructure depends on external systems
- Nothing depends on handlers or services
```

## Cleanup Worker State Machine

```
┌──────────────┐
│  Worker      │
│  Idle        │
│  (waiting)   │
└──────┬───────┘
       │
       │ 1 minute passes
       │
       ▼
┌──────────────────────┐
│ Check Redis          │
│ for expired leases   │
│ GetExpiredLeases()   │
└──────┬───────────────┘
       │
       ├─ No leases → Return to idle
       │
       └─ Found leases → Enter cleanup
                │
                ▼
        ┌──────────────────────┐
        │ For each lease:      │
        │ cleanupContainer()   │
        └──────┬───────────────┘
               │
        Retry Logic (max 3):
        ┌──────┴──────┐
        │ Attempt 1   │
        ├─ Try steps  │
        │ Success?    │
        │ ├─ YES → Done, log
        │ └─ NO → Wait & retry
        │ Attempt 2   │
        │ ├─ YES → Done, log
        │ └─ NO → Wait & retry
        │ Attempt 3   │
        │ ├─ YES → Done, log
        │ └─ NO → Log error
        └─────┬──────┘
              │
              ▼
    ┌──────────────────────┐
    │ Back to Idle,        │
    │ wait 1 minute        │
    │ (repeat forever)     │
    └──────────────────────┘
```

## Request/Response Cycle: Provisioning

```
CLIENT (React Frontend)
┌────────────────────────────┐
│ User clicks                │
│ "Provision Container"      │
│ selects: Ubuntu, 120 min   │
└──────────────┬─────────────┘
               │
               │ POST /api/provision
               │ Content-Type: application/json
               │ {
               │   "imageType": "ubuntu",
               │   "durationMinutes": 120
               │ }
               │
               ▼
BACKEND (Go Handler)
┌────────────────────────────┐
│ ProvisionHandler.ServeHTTP │
│ - Parse JSON              │
│ - Validate input          │
│ - Call ContainerService   │
└──────────────┬─────────────┘
               │
               ▼
SERVICE LAYER
┌────────────────────────────┐
│ ContainerService:          │
│ 1. Docker create           │
│    → "abc123" returned     │
│ 2. Repository.Save()       │
│ 3. Lease.CreateLease()     │
│    → TTL=7200s             │
│ 4. Return Container{}      │
└──────────────┬─────────────┘
               │
               ▼
RESPONSE
┌────────────────────────────┐
│ HTTP 201 Created           │
│ Content-Type: application/ │
│ json                       │
│ {                          │
│   "id": "abc123",          │
│   "expiryTime": "...",     │
│   "createdAt": "..."       │
│ }                          │
└──────────────┬─────────────┘
               │
               ▼
CLIENT
┌────────────────────────────┐
│ Frontend receives:         │
│ - Container ID             │
│ - Expiry time              │
│ - Updates state            │
│ - Shows in container list  │
│ - Starts countdown timer   │
│ - WebSocket for logs       │
└────────────────────────────┘
```

---

**These diagrams show the complete system architecture and data flows. Reference them when implementing components.**
