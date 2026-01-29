# Deployment Guide

## Overview

ContainerLease is deployed using Docker Compose for local development and can be scaled to Kubernetes for production.

## Quick Start (Local)

### Prerequisites
- Docker 24.0+
- Docker Compose 2.0+
- Git

### Environment Setup

1. Clone the repository:
```bash
git clone https://github.com/aryan0dhankhar/containerlease.git
cd containerlease
```

2. Create `.env` file in project root:
```bash
# Backend Configuration
POSTGRES_USER=containerlease
POSTGRES_PASSWORD=dev
POSTGRES_DB=containerlease_db
REDIS_URL=redis://redis:6379/0

# Frontend Configuration
VITE_API_URL=http://localhost:8080
```

3. Start services:
```bash
docker compose up -d --build
```

Services will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Services Architecture

### Backend (Go)
- **Port:** 8080
- **Runtime:** Go 1.24+
- **Dependencies:** PostgreSQL, Redis, Docker daemon

### Frontend (React + TypeScript)
- **Port:** 3000
- **Runtime:** Node.js 18+
- **Build:** Vite 7.3+

### Database (PostgreSQL)
- **Port:** 5432
- **Version:** 16-alpine
- **Persistence:** Docker volume `postgres_data`
- **Init:** Runs migrations automatically on startup

### Cache (Redis)
- **Port:** 6379
- **Version:** 7-alpine
- **Persistence:** Docker volume `redis_data`

## Configuration

### Backend Environment Variables
```
POSTGRES_USER=containerlease
POSTGRES_PASSWORD=<password>
POSTGRES_DB=containerlease_db
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
REDIS_URL=redis://redis:6379/0
LOG_LEVEL=info
```

### Frontend Environment Variables
```
VITE_API_URL=http://localhost:8080
```

## Database Migrations

Migrations run automatically on backend startup. To manage manually:

```bash
# Connect to database
docker exec -it containerlease-postgres-1 psql -U containerlease -d containerlease_db

# View migration files
ls backend/migrations/
```

## Production Deployment

### Kubernetes Deployment
See `deploy/k8s/` for Kubernetes manifests:
- `deployment.yaml` - Service deployments
- `service.yaml` - Service exposure
- `secret.yaml` - Environment secrets

Deploy to Kubernetes:
```bash
kubectl create namespace containerlease
kubectl apply -f deploy/k8s/secret.yaml -n containerlease
kubectl apply -f deploy/k8s/deployment.yaml -n containerlease
kubectl apply -f deploy/k8s/service.yaml -n containerlease
```

### Health Checks
Backend includes health endpoint for orchestration:
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/health
```

## Monitoring

### Logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f backend
docker compose logs -f frontend
```

### Metrics (Prometheus)
Backend exports Prometheus metrics at `/metrics` (requires auth token).

See `deploy/monitoring/prometheus-rules.yaml` for alerting rules.

## Troubleshooting

### Containers won't start
```bash
# Check logs
docker compose logs

# Rebuild images
docker compose down
docker compose up -d --build
```

### Database connection issues
```bash
# Test PostgreSQL connection
docker exec containerlease-postgres-1 pg_isready

# View database logs
docker compose logs postgres
```

### Frontend can't reach backend
- Verify backend is running: `docker compose ps`
- Check `VITE_API_URL` in frontend environment
- Ensure network connectivity: `docker network ls`

### Port conflicts
```bash
# Find process using port
lsof -i :8080
lsof -i :3000

# Change ports in docker-compose.yml and redeploy
```

## Shutdown

```bash
# Stop all services (preserve data)
docker compose down

# Stop and remove everything
docker compose down -v
```

## Security Considerations

1. **Environment Variables:** Never commit `.env` file
2. **Database Passwords:** Use strong, unique passwords in production
3. **CORS:** Configure allowed origins in backend
4. **Secrets:** Use Kubernetes secrets or environment management tool (Vault, 1Password)
5. **Network:** Use private networks, restrict port access
6. **TLS:** Terminate SSL at ingress layer (nginx, HAProxy, cloud load balancer)

## Performance Tuning

### Database
```yaml
# docker-compose.yml PostgreSQL
environment:
  POSTGRES_INITDB_ARGS: "-c shared_buffers=256MB -c effective_cache_size=1GB"
```

### Redis
- Use persistence sparingly
- Monitor memory usage
- Configure eviction policy

### Backend
- Adjust connection pools in `pkg/config/config.go`
- Enable caching for frequently accessed data
- Use circuit breaker for external API calls

## Backup & Recovery

### PostgreSQL Backup
```bash
docker exec containerlease-postgres-1 pg_dump -U containerlease containerlease_db > backup.sql
```

### Restore
```bash
docker exec -i containerlease-postgres-1 psql -U containerlease containerlease_db < backup.sql
```

## Version Management

Check current versions:
```bash
docker compose config | grep image
go version
npm --version
node --version
```

## Support

For issues and deployment questions, refer to:
- Backend: [ARCHITECTURE.md](ARCHITECTURE.md)
- API Docs: [API.md](API.md)
- Authentication: [AUTHENTICATION.md](AUTHENTICATION.md)
