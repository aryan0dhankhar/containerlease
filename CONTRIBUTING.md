# Contributing Guide

Thank you for contributing to ContainerLease! This guide covers development setup, code standards, and the pull request process.

## Development Environment Setup

### Prerequisites
- Go 1.24+
- Node.js 18+ with npm
- Docker 24.0+
- PostgreSQL 16+ (or use Docker)
- Redis 7+ (or use Docker)

### Quick Start

1. Clone and navigate to project:
```bash
git clone https://github.com/aryan0dhankhar/containerlease.git
cd containerlease
```

2. Start dependencies with Docker Compose:
```bash
docker compose up -d postgres redis
```

3. Set up backend:
```bash
cd backend
go mod download
go build -o bin/server ./cmd/server
```

4. Set up frontend:
```bash
cd ../frontend
npm install
npm run dev
```

5. Start development servers:
```bash
# Terminal 1: Backend
cd backend
go run ./cmd/server/main.go

# Terminal 2: Frontend
cd frontend
npm run dev
```

Access at http://localhost:3000

## Project Structure

```
containerlease/
├── backend/
│   ├── cmd/server/main.go          # Application entry point
│   ├── internal/
│   │   ├── domain/                 # Domain models
│   │   ├── handler/                # HTTP handlers
│   │   ├── service/                # Business logic
│   │   ├── repository/             # Data access
│   │   ├── infrastructure/         # External integrations
│   │   └── security/               # Auth & authorization
│   ├── migrations/                 # Database migrations
│   ├── pkg/                        # Shared packages
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/             # React components
│   │   ├── services/               # API clients
│   │   ├── styles/                 # CSS modules
│   │   └── types/                  # TypeScript types
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile
├── deploy/                         # Kubernetes & monitoring configs
├── docker-compose.yml
├── README.md
├── API.md
├── ARCHITECTURE.md
├── AUTHENTICATION.md
└── DEPLOYMENT.md
```

## Code Standards

### Go (Backend)

**Style Guide:** Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

```bash
# Format code
go fmt ./...

# Lint
golangci-lint run ./...

# Vet
go vet ./...
```

**Key Practices:**
- Use interfaces for abstractions
- Clean Architecture: domain → service → handler
- Error handling: explicit error returns, no panics in production code
- Testing: write tests for business logic

**Package Organization:**
- `internal/domain/` - Pure domain logic, no dependencies
- `internal/service/` - Business logic, uses repositories
- `internal/handler/` - HTTP request handling
- `internal/repository/` - Data access layer
- `internal/infrastructure/` - External integrations

### TypeScript/React (Frontend)

**Style Guide:** Follow [Airbnb React Style Guide](https://github.com/airbnb/javascript/tree/master/react)

```bash
# Format code
npm run format

# Lint
npm run lint

# Type check
npm run type-check
```

**Key Practices:**
- Use functional components with hooks
- Prop drilling: pass only needed props
- State management: React Context for global state
- API calls: centralize in `services/` directory
- Typing: strict TypeScript, avoid `any`

**Component Structure:**
```typescript
// src/components/MyComponent.tsx
import { FC } from 'react';
import styles from './MyComponent.css';

interface Props {
  title: string;
  onAction: () => void;
}

export const MyComponent: FC<Props> = ({ title, onAction }) => {
  return (
    <div className={styles.container}>
      <h1>{title}</h1>
      <button onClick={onAction}>Action</button>
    </div>
  );
};
```

## Testing

### Backend (Go)

```bash
cd backend

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./internal/service -run TestContainerService
```

Write tests in `*_test.go` files:
```go
func TestProvisionContainer(t *testing.T) {
  // Arrange
  service := NewContainerService(mockRepo, mockDocker)
  
  // Act
  container, err := service.Provision(ctx, req)
  
  // Assert
  assert.NoError(t, err)
  assert.NotNil(t, container)
}
```

### Frontend (React)

```bash
cd frontend

# Run tests (when available)
npm run test

# Run with coverage
npm run test:coverage
```

## Git Workflow

### Branch Naming
- Feature: `feature/description`
- Bug fix: `fix/description`
- Docs: `docs/description`

Example: `feature/add-container-logs`

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add container log streaming
fix: resolve race condition in cleanup worker
docs: update API documentation
refactor: simplify authorization logic
test: add unit tests for service layer
chore: update dependencies
```

### Pull Request Process

1. **Create branch from `main`:**
   ```bash
   git checkout -b feature/your-feature
   ```

2. **Make changes and test:**
   ```bash
   go test ./...
   npm run test
   npm run build
   ```

3. **Commit with conventional messages:**
   ```bash
   git add .
   git commit -m "feat: describe your change"
   ```

4. **Push to remote:**
   ```bash
   git push origin feature/your-feature
   ```

5. **Open Pull Request:**
   - Use clear title and description
   - Reference related issues: `Fixes #123`
   - Add screenshots for UI changes
   - Request reviewers

6. **Address feedback:**
   - Make requested changes
   - Push additional commits (don't rebase)
   - Respond to comments

7. **Merge:**
   - Squash commits to one for clean history
   - Delete branch after merge

## Documentation

- **Code comments:** Explain *why*, not *what*
- **Function docs:** Public functions should have comments
- **Complex logic:** Add inline comments
- **API changes:** Update [API.md](API.md)
- **Architecture changes:** Update [ARCHITECTURE.md](ARCHITECTURE.md)

Example Go doc:
```go
// ProvisionContainer creates a new temporary container with specified resources.
// It validates image allowlist, enforces resource limits, and schedules cleanup.
func (s *ContainerService) ProvisionContainer(ctx context.Context, req *ProvisionRequest) (*Container, error) {
  // ...
}
```

## Debugging

### Backend
```bash
# Run with verbose logging
LOG_LEVEL=debug go run ./cmd/server/main.go

# Run with debugger (dlv)
dlv debug ./cmd/server/main.go
```

### Frontend
- Open DevTools: F12
- Use React DevTools extension
- Check Network tab for API calls
- Use Console for errors

## Performance Considerations

- **Backend:** Minimize database queries, use indexes, cache frequently accessed data
- **Frontend:** Code splitting, lazy loading, memoization for expensive renders
- **Database:** Proper indexing, connection pooling, query optimization

## Security

- Never commit secrets (`.env` files, API keys)
- Use environment variables for configuration
- Validate all user input
- Follow OWASP guidelines
- Review security implications of changes

## Questions?

- Check existing [GitHub Issues](https://github.com/aryan0dhankhar/containerlease/issues)
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- Review [API.md](API.md) for endpoint details
- Check [AUTHENTICATION.md](AUTHENTICATION.md) for auth flows
