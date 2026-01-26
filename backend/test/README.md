# Integration Test Suite

## Overview
Comprehensive integration tests for ContainerLease platform covering all major features and endpoints.

## Prerequisites

1. **Running Services**: Ensure all services are running:
   ```bash
   docker compose up -d
   ```

2. **Wait for Readiness**: Give services ~10 seconds to start completely

## Running Tests

### All Tests
```bash
cd backend
go test ./test -v
```

### Specific Test
```bash
go test ./test -v -run TestHealthEndpoint
go test ./test -v -run TestProvisionContainer
go test ./test -v -run TestContainerLifecycle
```

### With Coverage
```bash
go test ./test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Benchmarks
```bash
go test ./test -bench=. -benchmem
```

## Test Categories

### 1. Health & Observability
- **TestHealthEndpoint**: Verifies `/healthz` returns OK
- **TestMetricsEndpoint**: Validates Prometheus metrics exposure
  - Checks for `containerlease_provision_duration_seconds`
  - Checks for `containerlease_cleanup_operations_total`
  - Checks for `containerlease_active_containers`

### 2. Configuration
- **TestPresetsEndpoint**: Validates preset templates
  - Ensures 3 presets (tiny, standard, large)
  - Verifies preset structure and IDs

### 3. Provisioning
- **TestProvisionContainer**: Tests basic container provisioning
  - Creates container with CPU/memory limits
  - Verifies async provisioning completes
  - Confirms container reaches "running" state
  - Includes automatic cleanup

- **TestProvisionContainerWithVolume**: Tests volume attachment
  - Provisions container with 512MB volume
  - Note: Full volume verification requires Docker API access

### 4. Validation
- **TestProvisionValidation**: Comprehensive input validation
  - Missing `imageType` → 400
  - Invalid `imageType` → 400
  - Duration out of bounds → 400
  - CPU/memory exceeding limits → 400
  - Volume size too large → 400

### 5. Lifecycle Management
- **TestContainerLifecycle**: End-to-end flow
  1. Provision container
  2. Wait for "running" state
  3. List containers
  4. Check status
  5. Delete container
  6. Verify "terminated" state

### 6. Concurrency
- **TestConcurrentProvisioning**: Parallel request handling
  - Sends 5 concurrent provision requests
  - Verifies all succeed
  - Tests system stability under load

### 7. Performance
- **BenchmarkProvision**: Measures provisioning throughput
  - Metrics: operations/sec, allocations, memory

## Test Outcomes

### Success Indicators
- All tests pass with `PASS` status
- No container leaks (cleanup successful)
- Response times < 5 seconds for provisioning
- Status endpoints return valid JSON

### Failure Scenarios
Tests will fail if:
- Services not running
- Ports already in use
- Redis unavailable
- Docker daemon not accessible
- Resource limits exceeded

## Cleanup Strategy

All tests use `t.Cleanup()` to ensure:
- Containers are deleted after tests
- No orphaned resources
- Clean state for subsequent runs

## Expected Output

```
=== RUN   TestHealthEndpoint
--- PASS: TestHealthEndpoint (0.01s)
=== RUN   TestMetricsEndpoint
--- PASS: TestMetricsEndpoint (0.02s)
=== RUN   TestPresetsEndpoint
--- PASS: TestPresetsEndpoint (0.01s)
=== RUN   TestProvisionContainer
    integration_test.go:XXX: Provisioned container: container-123456789
--- PASS: TestProvisionContainer (3.15s)
=== RUN   TestProvisionContainerWithVolume
    integration_test.go:XXX: Volume provisioning test completed
--- PASS: TestProvisionContainerWithVolume (3.12s)
=== RUN   TestProvisionValidation
=== RUN   TestProvisionValidation/missing_imageType
=== RUN   TestProvisionValidation/invalid_imageType
=== RUN   TestProvisionValidation/duration_too_low
=== RUN   TestProvisionValidation/duration_too_high
=== RUN   TestProvisionValidation/CPU_too_high
=== RUN   TestProvisionValidation/memory_too_high
=== RUN   TestProvisionValidation/volume_too_large
--- PASS: TestProvisionValidation (0.25s)
=== RUN   TestContainerLifecycle
    integration_test.go:XXX: Provisioned container: container-987654321
--- PASS: TestContainerLifecycle (4.23s)
=== RUN   TestConcurrentProvisioning
--- PASS: TestConcurrentProvisioning (3.45s)
PASS
ok      github.com/yourorg/containerlease/test  14.241s
```

## Benchmark Results

Example output:
```
BenchmarkProvision-8    10    115234567 ns/op    24576 B/op    125 allocs/op
```

Interpretation:
- **10 iterations**: Test ran 10 times
- **115ms/op**: Average provisioning time
- **24KB/op**: Memory allocated per operation
- **125 allocs/op**: Memory allocations per operation

## Troubleshooting

### "connection refused"
**Cause**: Services not running  
**Fix**: `docker compose up -d`

### "container not found"
**Cause**: Async provisioning not complete  
**Fix**: Tests include 3-second wait; adjust if needed

### "rate limit exceeded"
**Cause**: Too many requests (>100/min)  
**Fix**: Wait 60 seconds before re-running tests

### "volume verification skipped"
**Cause**: Tests run via HTTP, no Docker API access  
**Fix**: This is expected; use `docker volume ls` manually

## Adding New Tests

### Pattern
```go
func TestMyFeature(t *testing.T) {
    // 1. Setup
    payload := `{"key": "value"}`
    
    // 2. Execute
    resp, err := http.Post(baseURL+"/api/endpoint", "application/json", strings.NewReader(payload))
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // 3. Assert
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
    
    // 4. Cleanup
    t.Cleanup(func() {
        // cleanup resources
    })
}
```

### Best Practices
1. **Use t.Cleanup()**: Ensures cleanup even on failure
2. **Wait for async**: Add delays after provisioning (3s recommended)
3. **Descriptive names**: Test names should explain what they verify
4. **Independent tests**: Each test should run in isolation
5. **Idempotent**: Tests should be re-runnable without side effects

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v2
      - name: Start services
        run: docker compose up -d
      - name: Wait for readiness
        run: sleep 15
      - name: Run tests
        run: |
          cd backend
          go test ./test -v
      - name: Cleanup
        run: docker compose down
```

## Coverage Goals

Current test coverage targets:
- **Handlers**: 80%+ (HTTP endpoint logic)
- **Service Layer**: 70%+ (business logic)
- **Infrastructure**: 60%+ (external integrations)

To check coverage:
```bash
go test ./test -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Future Test Additions

### Planned Tests
1. **WebSocket Logs**: Test log streaming
2. **Authentication**: JWT token validation
3. **Rate Limiting**: 100 req/min enforcement
4. **Cleanup Worker**: GC reconciliation logic
5. **Circuit Breaker**: Docker API failure scenarios
6. **Volume Lifecycle**: Full volume CRUD with Docker API
7. **Billing**: Cost calculation accuracy
8. **Metrics**: Counter/gauge/histogram validation

### Performance Tests
1. Load testing (100+ concurrent users)
2. Stress testing (resource exhaustion)
3. Soak testing (24hr stability)
4. Spike testing (sudden traffic bursts)

## Notes

- Tests use `alpine` image by default (faster pulls)
- Default test duration: 10 minutes (long enough to verify, short cleanup)
- Endpoints are public for testing (JWT auth skipped)
- Cleanup happens automatically via `t.Cleanup()`
- Failed tests leave containers in "terminated" state for debugging
