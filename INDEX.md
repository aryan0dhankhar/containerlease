# ContainerLease - Documentation Index

## 📖 Getting Started

Start here if you're new to the project:

1. **[SUMMARY.md](SUMMARY.md)** ⭐ START HERE
   - Quick project overview
   - Status & next steps
   - Key locations & important notes

2. **[README.md](README.md)**
   - Complete feature documentation
   - API endpoints
   - Tech stack details
   - Quick start instructions

## 🏗️ Architecture & Design

Understanding the system design:

3. **[ARCHITECTURE.md](ARCHITECTURE.md)** ⭐ CRITICAL
   - Deep dive into cleanup worker logic
   - Why separate background worker?
   - Data flow & Redis schema
   - Error handling strategy
   - Scalability considerations

4. **[PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)**
   - Complete directory tree
   - Explanation of each layer
   - Where cleanup logic resides
   - Tech stack summary

5. **[DIAGRAMS.md](DIAGRAMS.md)**
   - System architecture diagram
   - Cleanup worker sequence diagram
   - Data flow diagrams
   - Clean architecture dependency flow
   - State machines

## 📋 Implementation

Step-by-step guide for building the project:

6. **[IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)** ⭐ FOR BUILDERS
   - Phase 1-6 implementation roadmap
   - Detailed tasks with status
   - Code standards checklist
   - Debugging tips
   - Quick command reference

## 🗂️ Directory Structure

```
containerlease/
├── README.md                          # Start here (overview)
├── SUMMARY.md                         # Quick summary
├── ARCHITECTURE.md                    # Cleanup logic deep dive
├── PROJECT_STRUCTURE.md               # Directory explanation
├── DIAGRAMS.md                        # Visual diagrams
├── IMPLEMENTATION_GUIDE.md            # Implementation roadmap
├── docker-compose.yml                 # Local dev setup
├── .gitignore
│
├── backend/                           # Go backend
│   ├── cmd/server/main.go             # Entry point
│   ├── internal/
│   │   ├── domain/                    # Entities & interfaces
│   │   ├── handler/                   # HTTP handlers
│   │   ├── service/                   # Business logic
│   │   ├── repository/                # Data access
│   │   ├── infrastructure/            # External clients
│   │   ├── middleware/                # HTTP middleware
│   │   └── worker/                    # Background jobs ⭐
│   │       └── cleanup_worker.go      # Cleanup logic ⭐⭐⭐
│   ├── pkg/                           # Reusable packages
│   ├── config/                        # Configuration
│   ├── go.mod
│   └── Dockerfile
│
└── frontend/                          # React frontend
    ├── src/
    │   ├── components/                # React components
    │   ├── hooks/                     # Custom hooks
    │   ├── services/                  # API client
    │   ├── types/                     # TypeScript types
    │   └── App.tsx
    ├── vite.config.ts
    ├── tsconfig.json
    ├── package.json
    └── Dockerfile
```

## 🎯 Key Files to Understand

### For Understanding Cleanup Logic
1. **[backend/internal/worker/cleanup_worker.go](backend/internal/worker/cleanup_worker.go)** ⭐⭐⭐
   - The MOST IMPORTANT file
   - Background worker running every 1 minute
   - Automatic container cleanup

2. **[backend/internal/service/container_service.go](backend/internal/service/container_service.go)**
   - Container provisioning logic
   - Lease creation
   - Error handling

3. **[backend/cmd/server/main.go](backend/cmd/server/main.go)**
   - Application entry point
   - Worker initialization
   - Server startup

### For Understanding Architecture
1. **[backend/internal/domain/container.go](backend/internal/domain/container.go)**
   - Domain entities
   - Interface contracts
   - Type definitions

2. **[backend/internal/repository/lease_repository.go](backend/internal/repository/lease_repository.go)**
   - Redis integration
   - TTL management
   - Lease expiry queries

### For Frontend
1. **[frontend/src/components/ProvisionForm.tsx](frontend/src/components/ProvisionForm.tsx)**
   - Container provisioning UI
   - Form handling
   - Error display

2. **[frontend/src/services/containerApi.ts](frontend/src/services/containerApi.ts)**
   - REST API client
   - WebSocket management
   - API integration

## 📚 Reading Order

### Quick Start (30 min)
```
1. SUMMARY.md (overview)
2. README.md (features & tech stack)
3. docker-compose up (run locally)
```

### Understanding the System (2 hours)
```
1. ARCHITECTURE.md (cleanup logic)
2. DIAGRAMS.md (visual understanding)
3. Read key implementation files:
   - cleanup_worker.go
   - container_service.go
   - lease_repository.go
```

### Building the Project (ongoing)
```
1. IMPLEMENTATION_GUIDE.md (phase by phase)
2. Start Phase 1 (Docker client)
3. Test cleanup worker
4. Implement remaining handlers
5. Build frontend
```

## 🔍 Quick Reference

### Where is X?

**Cleanup Logic?**
→ [backend/internal/worker/cleanup_worker.go](backend/internal/worker/cleanup_worker.go)

**Container Provisioning?**
→ [backend/internal/service/container_service.go](backend/internal/service/container_service.go)

**HTTP Handlers?**
→ [backend/internal/handler/](backend/internal/handler/)

**Docker Integration?**
→ [backend/internal/infrastructure/docker/client.go](backend/internal/infrastructure/docker/client.go)

**Redis Integration?**
→ [backend/internal/infrastructure/redis/client.go](backend/internal/infrastructure/redis/client.go)

**Frontend Components?**
→ [frontend/src/components/](frontend/src/components/)

**API Types?**
→ [frontend/src/types/](frontend/src/types/)

### How do I...

**Run the project locally?**
```bash
cd containerlease
docker-compose up
```
→ See [README.md](README.md#quick-start)

**Understand cleanup logic?**
→ Read [ARCHITECTURE.md](ARCHITECTURE.md#cleanup-logic---the-heart-of-containerlease)

**Implement missing features?**
→ Follow [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)

**See visual diagrams?**
→ Check [DIAGRAMS.md](DIAGRAMS.md)

**Understand code structure?**
→ Read [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)

## 🎓 Learning Path

### For Systems Architects
1. SUMMARY.md
2. ARCHITECTURE.md (full read)
3. DIAGRAMS.md (study all diagrams)
4. Understand cleanup_worker.go implementation

### For Go Developers
1. README.md (tech stack)
2. PROJECT_STRUCTURE.md (layer explanation)
3. backend/internal/domain/container.go (entities)
4. IMPLEMENTATION_GUIDE.md Phase 1-3
5. Implement Docker client, repositories, handlers

### For React Developers
1. README.md (overview)
2. frontend/src/types/container.ts (types)
3. frontend/src/services/containerApi.ts (API calls)
4. IMPLEMENTATION_GUIDE.md Phase 4
5. Implement React components

### For DevOps/Infrastructure
1. docker-compose.yml (local setup)
2. backend/Dockerfile (backend build)
3. frontend/Dockerfile (frontend build)
4. IMPLEMENTATION_GUIDE.md Phase 6
5. Prepare deployment configs

## ✅ Completion Checklist

Use this to track your progress:

- [ ] Read SUMMARY.md
- [ ] Read README.md
- [ ] Understand ARCHITECTURE.md
- [ ] Study DIAGRAMS.md
- [ ] Phase 1: Implement Docker client
- [ ] Phase 2: Test cleanup worker
- [ ] Phase 2: Implement handlers
- [ ] Phase 3: Test cleanup end-to-end
- [ ] Phase 4: Build frontend components
- [ ] Phase 5: Add tests
- [ ] Phase 6: Deploy to production

## 📞 Navigation Tips

### From Any Documentation File
- All links use relative paths
- Markdown files use [filename](filename.md) format
- Code files use [path/file.go](path/file.go) format

### Key Links Always Available
- [SUMMARY.md](SUMMARY.md) - Overview
- [ARCHITECTURE.md](ARCHITECTURE.md) - Deep dive
- [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) - Build roadmap

## 🚀 Next Steps

1. **Read [SUMMARY.md](SUMMARY.md)** for 5-minute overview
2. **Read [README.md](README.md)** for complete picture
3. **Study [ARCHITECTURE.md](ARCHITECTURE.md)** for cleanup logic
4. **Review [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)** for next steps
5. **Start Phase 1** - Implement Docker client

---

**Version:** 1.0  
**Created:** January 15, 2026  
**Status:** Ready for implementation
