# User Login & Rate Limiting Implementation

## Overview
ContainerLease now includes a complete authentication and rate limiting system protecting all API endpoints with tenant isolation and brute-force protection.

---

## Features Added

### 1. User Authentication System

#### Login Endpoint
- **Route**: `POST /api/login`
- **Public**: Yes (no auth required)
- **Rate Limit**: 10 attempts per 5 minutes per IP address

#### Request
```json
{
  "email": "demo@example.com",
  "password": "demo123"
}
```

#### Response
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2026-01-27T12:34:56Z",
  "tenantId": "tenant-demo",
  "userId": "user-demo-1"
}
```

#### Demo Users
Three demo accounts are pre-configured for testing:

| Email | Password | Tenant |
|-------|----------|--------|
| demo@example.com | demo123 | tenant-demo |
| admin@example.com | admin123 | tenant-admin |
| test@example.com | test123 | tenant-test |

### 2. JWT Token-Based Authorization

All API endpoints (except login, presets, health) require a valid JWT token.

#### Using the Token
Add the token to the Authorization header:
```
Authorization: Bearer <token>
```

#### Token Claims
```json
{
  "tenant_id": "tenant-demo",
  "user_id": "user-demo-1",
  "email": "demo@example.com",
  "iat": 1643846400,
  "exp": 1643932800,
  "iss": "containerlease"
}
```

#### Token Expiration
- Default: 24 hours
- Generated on login
- Validated on every protected request

### 3. Advanced Rate Limiting

#### Tenant-Based Rate Limiting
- **Limit**: 100 requests per minute per tenant
- **Window**: 1 minute sliding window
- **Identification**: Tenant ID from JWT claims

#### IP-Based Rate Limiting (Unauthenticated)
- For public endpoints (login, presets)
- Identifies clients by IP address
- Supports X-Forwarded-For header (proxies/load balancers)

#### Login Brute-Force Protection
- **Limit**: 10 attempts per 5 minutes per IP address
- **Response**: 429 Too Many Requests
- **Prevents**: Dictionary attacks, credential stuffing

#### Rate Limit Response
```json
{
  "error": "rate limit exceeded"
}
```
HTTP Status: 429 Too Many Requests

### 4. Frontend Login Component

#### LoginForm Component
- Professional login UI with gradient background
- Email and password fields
- Error message display
- Demo credentials quick-select buttons
- Fully responsive design

#### Location
`frontend/src/components/LoginForm.tsx`

#### Features
- Form validation
- Loading state during authentication
- Automatic token persistence to localStorage
- Demo credentials for testing
- Beautiful CSS styling

#### Integration
- Automatically shown when no token exists
- Persists login state across page reloads
- Logout button in app header

---

## Architecture

### Authentication Flow
```
1. User enters credentials on LoginForm
2. Frontend sends POST /api/login
3. Backend validates credentials against UserStore
4. TokenManager generates JWT token
5. Token stored in localStorage on client
6. Token sent in Authorization header for all requests
7. JWTMiddleware validates on protected endpoints
```

### Rate Limiting Flow
```
1. Request arrives at RateLimitMiddleware
2. Skip check for /healthz, /readyz, /metrics
3. For login endpoint: IP-based limit (10/5min)
4. For other endpoints: Tenant-based limit (100/min)
5. If limit exceeded: 429 response
6. Otherwise: increment counter and continue
7. Cleanup: stale buckets removed after 15 minutes
```

### Middleware Chain
```
Request → Request ID → Rate Limit → JWT Auth → Audit → Handler
                         ↓
                    (Strict for /api/login)
                    (Standard for others)
```

---

## API Security

### Public Endpoints
- `/api/login` - User authentication
- `/api/presets` - Container presets
- `/healthz` - Health check
- `/readyz` - Readiness check
- `/metrics` - Prometheus metrics

### Protected Endpoints
All other endpoints require `Authorization: Bearer <token>` header:
- `POST /api/provision` - Create containers
- `GET /api/containers` - List containers
- `GET /api/containers/{id}/status` - Check status
- `DELETE /api/containers/{id}` - Delete container
- `GET /ws/logs/{id}` - WebSocket logs

---

## Configuration

### Environment Variables
```bash
# JWT Configuration
JWT_SECRET="your-secret-key-change-in-production"

# Rate Limiting (in main.go)
100 requests per minute per tenant
10 login attempts per 5 minutes per IP
```

### User Management
- Located in `internal/security/auth/users.go`
- In-memory store (demo implementation)
- Add users via `NewUserStore()` constructor
- **Production**: Replace with database (PostgreSQL, etc.)

---

## Testing

### Login Example
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@example.com",
    "password": "demo123"
  }'
```

### Authenticated Request Example
```bash
# First, get the token from login
TOKEN=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo123"}' | jq -r '.token')

# Use token to provision a container
curl -X POST http://localhost:8080/api/provision \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "imageType": "alpine",
    "durationMinutes": 30,
    "cpuMilli": 500,
    "memoryMB": 512
  }'
```

### Rate Limit Test
```bash
# Send 11 rapid login requests (should fail on 11th)
for i in {1..11}; do
  curl -X POST http://localhost:8080/api/login \
    -H "Content-Type: application/json" \
    -d '{"email":"demo@example.com","password":"demo123"}' \
    -w "\n[Request $i] Status: %{http_code}\n"
done
# Last request returns 429 Too Many Requests
```

---

## Security Best Practices Implemented

✅ **Password Security**
- Hashed with SHA-256 (for demo)
- **Production**: Use bcrypt or argon2

✅ **Token Security**
- HS256 HMAC signing
- 24-hour expiration
- Validated on every request

✅ **Brute Force Protection**
- 10 login attempts per 5 minutes per IP
- Generic error message ("invalid credentials")
- Prevents user enumeration

✅ **Tenant Isolation**
- Rate limits per tenant
- Token claims include tenant ID
- Audit logging tracks all actions

✅ **IP Protection**
- X-Forwarded-For header support
- IP-based rate limiting for public endpoints
- Works behind proxies and load balancers

---

## Production Checklist

- [ ] Change `JWT_SECRET` environment variable
- [ ] Replace UserStore with database (PostgreSQL)
- [ ] Use bcrypt or argon2 for password hashing
- [ ] Enable HTTPS/TLS
- [ ] Review rate limits for your use case
- [ ] Set up token refresh mechanism (optional)
- [ ] Configure CORS allowed origins properly
- [ ] Monitor audit logs for suspicious activity
- [ ] Set up log aggregation (ELK, Datadog, etc.)
- [ ] Regular security audits

---

## Frontend Integration

### Automatic Token Management
```typescript
// Token automatically added to all requests
const token = localStorage.getItem('token')
headers.Authorization = `Bearer ${token}`
```

### Login Persistence
```typescript
// Token stored after successful login
localStorage.setItem('token', response.token)
localStorage.setItem('user', JSON.stringify(user))

// Checked on app load
const token = localStorage.getItem('token')
if (token) {
  setIsAuthenticated(true)
}
```

### Logout Handling
```typescript
// Clear stored data
localStorage.removeItem('token')
localStorage.removeItem('user')
// Redirect to login
setIsAuthenticated(false)
```

---

## Metrics & Monitoring

### Rate Limiting Metrics
- `containerlease_rate_limit_exceeded_total` - Counter of rate limit violations
- `containerlease_login_attempts_total` - Counter of login attempts
- `containerlease_authentication_failures_total` - Counter of failed authentications

### Audit Logging
All authentication and authorization events logged:
```
action=login user_id=user-demo-1 tenant_id=tenant-demo status=success
action=login user_id=unknown status=failed reason=invalid_password
action=rate_limit_exceeded tenant_id=tenant-demo reason=api_limit
```

---

## Troubleshooting

### "Missing Auth" Error
**Problem**: `{"error":"missing auth"}`
**Solution**: Add `Authorization: Bearer <token>` header to request

### "Invalid Token" Error
**Problem**: `{"error":"invalid token"}`
**Solution**: 
- Token may be expired (24h lifetime)
- Re-login to get new token
- Check JWT_SECRET hasn't changed

### "Rate Limit Exceeded" Error
**Problem**: `{"error":"rate limit exceeded"}`
**Solution**:
- Wait 1 minute before retrying
- For login: wait 5 minutes for IP-based limit

### Login Not Working
**Problem**: Always shows "invalid credentials"
**Solution**:
- Verify email matches exactly
- Check password is correct
- Use demo credentials: demo@example.com / demo123

---

## Future Enhancements

- [ ] Token refresh endpoints (refresh token rotation)
- [ ] Social login (OAuth 2.0, Google, GitHub)
- [ ] Two-factor authentication (2FA)
- [ ] LDAP/Active Directory integration
- [ ] Role-based access control (RBAC)
- [ ] API key authentication for service-to-service
- [ ] Session management with multiple devices
- [ ] Password reset flow
- [ ] Account lockout after failed attempts
- [ ] Webhook notifications for security events
