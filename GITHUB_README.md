# ContainerLease - Ephemeral Dev Environment Portal

A modern web platform for provisioning temporary Docker containers with automatic cleanup. Perfect for testing code, demo environments, or ephemeral workloads.

![Status](https://img.shields.io/badge/status-functional-green)
![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

## 🚀 Features

- **One-Click Provisioning**: Spin up containers (Alpine/Ubuntu) in seconds
- **Automatic Lifecycle Management**: Containers auto-delete after lease expires
- **Real-Time Countdown**: See exactly when your container will be destroyed
- **Manual Control**: Delete containers before expiry if needed
- **Background Cleanup**: Garbage collector runs every minute to clean expired resources
- **Live Logs**: Stream container logs in real-time via WebSocket
- **CORS-Enabled**: Frontend and backend communicate seamlessly

## 📊 Architecture

```
┌─────────────────────────────────────┐
│     React Frontend (Port 3000)      │
│   Provision Form + Container List   │
└──────────────┬──────────────────────┘
               │ HTTP/WebSocket
               ↓
┌──────────────────────────────────────┐
│   Go Backend (Port 8080)             │
├──────────────────────────────────────┤
│  POST /api/provision                 │
│  GET /api/containers                 │
│  DELETE /api/containers/{id}         │
│  GET /ws/logs/{id}                   │
├──────────────────────────────────────┤
│  Cleanup Worker (runs every 1 min)   │
└──────────────┬──────────┬────────────┘
               │          │
               ↓          ↓
          Docker       Redis
         (Port 2375)   (Port 6379)
```

## ⚡ Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.24+
- Node.js 20+

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/containerlease.git
cd containerlease

# Start all services
docker compose up -d

# Services will be available at:
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
# Redis: localhost:6379
```

### Usage

1. **Open Frontend**: Visit http://localhost:3000
2. **Provision a Container**: 
   - Select image (Alpine/Ubuntu)
   - Set duration (5-480 minutes)
   - Click "Provision Container"
3. **Monitor**: Watch the countdown timer
4. **Delete**: Click delete button or let cleanup worker auto-delete at expiry

## 📋 API Endpoints

### Provision Container
```bash
POST /api/provision
Content-Type: application/json

{
  "imageType": "alpine",
  "durationMinutes": 120
}

Response (201 Created):
{
  "id": "abc123...",
  "imageType": "alpine",
  "expiryTime": "2026-01-16T18:20:00Z",
  "createdAt": "2026-01-16T16:20:00Z"
}
```

### List Containers
```bash
GET /api/containers

Response (200 OK):
{
  "containers": [
    {
      "id": "abc123...",
      "imageType": "alpine",
      "status": "running",
      "createdAt": "2026-01-16T16:20:00Z",
      "expiryAt": "2026-01-16T18:20:00Z",
      "expiresIn": 7200
    }
  ]
}
```

### Delete Container
```bash
DELETE /api/containers/{id}

Response (204 No Content)
```

### Stream Logs (WebSocket)
```bash
WS ws://localhost:8080/ws/logs/{id}
```

## 🏗️ Project Structure

```
containerlease/
├── backend/                    # Go server
│   ├── cmd/server/            # Entry point
│   ├── internal/
│   │   ├── domain/            # Business entities
│   │   ├── handler/           # HTTP handlers
│   │   ├── service/           # Business logic
│   │   ├── repository/        # Data access (Redis)
│   │   ├── infrastructure/    # Docker, Redis, Logger
│   │   ├── middleware/        # Middleware
│   │   └── worker/            # Cleanup worker
│   ├── pkg/config/            # Configuration
│   ├── go.mod & go.sum
│   ├── Dockerfile
│   └── server (binary)
│
├── frontend/                   # React + TypeScript
│   ├── src/
│   │   ├── components/        # React components
│   │   ├── services/          # API client
│   │   ├── types/             # TypeScript interfaces
│   │   ├── App.tsx            # Main app
│   │   └── main.tsx           # Entry point
│   ├── package.json
│   ├── Dockerfile
│   └── dist/ (build output)
│
├── docker-compose.yml          # Orchestration
├── README.md                   # This file
└── .gitignore
```

## 🔧 Configuration

### Backend Environment Variables
```bash
REDIS_URL=redis://redis:6379
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_API_VERSION=1.44
SERVER_PORT=8080
CLEANUP_INTERVAL_MINUTES=1
LOG_LEVEL=debug
```

### Frontend Configuration
API endpoints are configured in `frontend/src/services/containerApi.ts`:
```typescript
const BACKEND_URL = 'http://localhost:8080'
```

## 🧪 Testing

### Create a Container via CLI
```bash
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -d '{"imageType":"alpine","durationMinutes":5}'
```

### List Active Containers
```bash
curl http://localhost:8080/api/containers | jq
```

### Delete a Container
```bash
curl -X DELETE http://localhost:8080/api/containers/{container_id}
```

## 🚧 Roadmap

- [ ] User Authentication (OAuth via Google/GitHub)
- [ ] Cost Calculator (track compute costs)
- [ ] Multi-instance types (Small/Medium/Large)
- [ ] Container Exec API (run commands in containers)
- [ ] Usage Dashboard (metrics, history)
- [ ] SSH/Shell Access
- [ ] Cloud VM Support (AWS EC2, DigitalOcean)
- [ ] Email Notifications (expiry warnings)

## 🛠️ Development

### Build Backend
```bash
cd backend
go build ./cmd/server
./server
```

### Build Frontend
```bash
cd frontend
npm install
npm run build
npm run preview  # Preview production build
```

### Run Tests
```bash
cd backend
go test ./...
```

## 📦 Technologies

- **Backend**: Go 1.25, Gorilla WebSocket
- **Frontend**: React 18, TypeScript, Vite
- **Storage**: Redis (TTL-based)
- **Container**: Docker & Docker Compose
- **API Style**: RESTful with WebSocket for logs

## 🔒 Security Considerations

- ⚠️ **No Authentication**: Currently open to anyone. Add OAuth before production use.
- ⚠️ **No Rate Limiting**: Add rate limits to prevent abuse.
- ⚠️ **CORS Allow-All**: Change `Access-Control-Allow-Origin: *` for production.
- ⚠️ **No Input Validation**: Sanitize image names and duration inputs.

## 📝 License

MIT License - see LICENSE file for details

## 🤝 Contributing

Contributions welcome! Please:
1. Fork the repo
2. Create a feature branch
3. Make changes and test
4. Submit a pull request

## 📧 Support

For issues, questions, or feature requests, open a GitHub issue.

---

**Status**: ✅ Core functionality complete and tested. Ready for portfolio/demo.
