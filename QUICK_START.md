# ContainerLease - Quick Start Guide

## 🚀 Get Started in 3 Minutes

### Prerequisites
```bash
# Install Redis (macOS)
brew install redis

# Install Node.js (macOS)
brew install node

# Already installed: Go 1.24+, Docker
```

### Option 1: Local Development (Recommended for Testing)

#### Terminal 1: Start Redis
```bash
redis-server --port 6379 --daemonize yes
redis-cli ping  # Should output: PONG
```

#### Terminal 2: Start Backend
```bash
cd /Users/aryandhankhar/Documents/dev/containerlease/backend
REDIS_URL="redis://localhost:6379" go run ./cmd/server

# Expected output:
# {"time":"...","level":"INFO","msg":"starting ContainerLease server"...}
# {"time":"...","level":"INFO","msg":"server starting","port":8080}
```

#### Terminal 3: Start Frontend
```bash
cd /Users/aryandhankhar/Documents/dev/containerlease/frontend
npm run dev

# Expected output:
# VITE v5.4.21  ready in 191 ms
# ➜  Local:   http://localhost:5173/
```

#### Terminal 4: Test the API
```bash
# List containers (should be empty initially)
curl http://localhost:8080/api/containers

# Provision a container (2-minute lifetime)
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":2}'

# List containers again (should show your new container)
curl http://localhost:8080/api/containers
```

#### Terminal 5: View in Browser
```
Open http://localhost:5173
- See Provision Form
- See Active Containers list
- Watch countdown timer
- Click Delete to remove manually
```

---

### Option 2: Docker Compose (Production)

```bash
cd /Users/aryandhankhar/Documents/dev/containerlease

# Start all services
docker compose up

# In another terminal:
# Test API
curl http://localhost:8080/api/containers

# View in browser
# http://localhost:3000
```

---

## 📋 API Endpoints

### List Containers
```bash
curl http://localhost:8080/api/containers
```

### Provision Container
```bash
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":30}'

# Available images: "alpine", "ubuntu"
# Duration: 5-480 minutes
```

### Delete Container (Manual)
```bash
curl -X DELETE http://localhost:8080/api/containers/{CONTAINER_ID}
```

### Stream Logs (WebSocket)
```bash
# JavaScript
const ws = new WebSocket('ws://localhost:8080/ws/logs/{CONTAINER_ID}');
ws.onmessage = (e) => console.log(e.data);
```

---

## 🔍 What Happens When You Provision

1. **Click "Provision Container"** → Form submits to backend
2. **Backend receives** → Validates input
3. **Docker pulls image** → Downloads Alpine/Ubuntu (60-120s first time)
4. **Container created** → With 512MB memory limit
5. **Container started** → Running `sleep infinity`
6. **Container saved** → To Redis with TTL
7. **Response sent** → Frontend shows container ID

8. **Cleanup timer** → Every 1 minute, worker checks Redis
9. **When expired** → Worker stops and removes container
10. **Cleanup done** → Container disappears from list

---

## 📁 Where Things Are

| What | Where |
|------|-------|
| Backend code | `/backend/internal/` |
| Frontend code | `/frontend/src/` |
| Docker setup | `/docker-compose.yml` |
| Configuration | `backend/pkg/config/config.go` |
| Cleanup worker | `backend/internal/worker/cleanup_worker.go` |
| API handlers | `backend/internal/handler/*.go` |
| React components | `frontend/src/components/*.tsx` |

---

## 🧪 Test Scenarios

### Test 1: Provision and Watch Auto-Cleanup
```bash
# 1. Provision with 1-minute TTL
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":1}'

# 2. List containers immediately (should see it)
curl http://localhost:8080/api/containers

# 3. Wait 1-2 minutes

# 4. List again (should be gone)
curl http://localhost:8080/api/containers

# ✓ Auto-cleanup works!
```

### Test 2: Manual Deletion
```bash
# 1. Provision a container
RESPONSE=$(curl -s -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":30}')
ID=$(echo $RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

# 2. Delete it manually
curl -X DELETE http://localhost:8080/api/containers/$ID

# 3. Verify it's gone
curl http://localhost:8080/api/containers

# ✓ Manual deletion works!
```

### Test 3: Frontend Integration
1. Open `http://localhost:5173` in browser
2. Fill in Provision Form:
   - Image: Alpine
   - Duration: 5 minutes
3. Click "Provision Container"
4. Watch container appear in list
5. See countdown timer counting down
6. Click "Delete" to remove manually
7. Watch cleanup worker remove after 5 minutes

---

## ⚡ Performance Notes

- **First provision**: 60-120s (includes image pull)
- **Subsequent provisions**: 5-10s (image cached)
- **List refresh**: Every 5 seconds
- **Timer update**: Every 1 second
- **Cleanup check**: Every 1 minute
- **Memory per container**: ~50-100MB

---

## 🐛 Troubleshooting

### Backend won't start
```bash
# Check Redis is running
redis-cli ping

# Check port 8080 is free
lsof -i :8080

# View full error
REDIS_URL="redis://localhost:6379" go run ./cmd/server
```

### Frontend won't load
```bash
# Check node_modules
cd frontend && npm install

# Clear cache
rm -rf node_modules/.vite

# Rebuild
npm run build
```

### Containers won't provision
```bash
# Check Docker is running
docker ps

# Check image pulled
docker images | grep alpine

# Check logs
# Look at backend terminal for error messages
```

### WebSocket not connecting
```bash
# Check protocol - should be ws:// not http://
# Check browser console for connection errors
# Verify backend WebSocket handler at /ws/logs/{id}
```

---

## 📊 Monitoring Commands

```bash
# Watch active containers
watch curl -s http://localhost:8080/api/containers | jq '.containers | length'

# Watch Redis keys
watch redis-cli keys '*'

# Watch Docker containers
watch docker ps

# Watch process memory
watch ps aux | grep server

# Check logs in real-time (Terminal 2)
# Just watch the terminal where backend is running
```

---

## 🎯 Next Steps

1. ✅ Explore the code in `/backend` and `/frontend`
2. ✅ Try provisioning different containers
3. ✅ Test the cleanup worker by watching auto-deletion
4. ✅ Check the documentation in root folder
5. ✅ Deploy with `docker compose up` when ready

---

## 📚 Documentation

- **PROJECT_COMPLETION_REPORT.md** - Full project details
- **ARCHITECTURE.md** - System design
- **IMPLEMENTATION_GUIDE.md** - Feature details
- **README.md** - Overview
- **PHASE4_STATUS.md** - Testing notes

---

**Need help?** Check the terminal output - all errors are logged with details!
