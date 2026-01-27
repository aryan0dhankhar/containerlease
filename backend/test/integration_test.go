package test

import (
	"io"
	"net/http"
	"testing"
)

// TestHealthEndpoint verifies health check endpoint
func TestHealthEndpoint(t *testing.T) {
	server := NewTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL() + "/healthz")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" && string(body) != "OK" {
		t.Errorf("Expected 'ok' or 'OK', got '%s'", string(body))
	}
}

// TestReadinessEndpoint verifies readiness check endpoint
func TestReadinessEndpoint(t *testing.T) {
	server := NewTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL() + "/readyz")
	if err != nil {
		t.Fatalf("Readiness check failed: %v", err)
	}
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ready" {
		t.Errorf("Expected 'ready', got '%s'", string(body))
	}
}

// TestMetricsEndpoint verifies Prometheus metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	server := NewTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL() + "/metrics")
	if err != nil {
		t.Fatalf("Metrics endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	AssertStatusCode(t, resp, http.StatusOK)
	AssertContentType(t, resp, "text/plain")

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	// Just verify we got metrics back (even in test it should be minimal)
	if len(metrics) < 1 {
		t.Errorf("Expected metrics data, got empty response")
	}
}

// TestPresetsEndpoint verifies presets configuration
func TestPresetsEndpoint(t *testing.T) {
	t.Skip("Integration test requires running server - use docker-compose up")
}

// TestProvisionContainer verifies container provisioning flow
func TestProvisionContainer(t *testing.T) {
	t.Skip("Integration test requires running server - use docker-compose up")
}

// TestProvisionContainerWithVolume verifies volume provisioning
func TestProvisionContainerWithVolume(t *testing.T) {
	t.Skip("Integration test requires running server - use docker-compose up")
}

// TestProvisionValidation verifies input validation
func TestProvisionValidation(t *testing.T) {
	t.Skip("Integration test requires running server - use docker-compose up")
}
