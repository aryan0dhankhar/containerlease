package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	baseURL       = "http://localhost:8080"
	testImageType = "alpine"
	testDuration  = 10
)

// TestHealthEndpoint verifies health check endpoint
func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		// Accept both "OK" and "ok"
		if string(body) != "ok" {
			t.Errorf("Expected 'ok' or 'OK', got '%s'", string(body))
		}
	}
}

// TestMetricsEndpoint verifies Prometheus metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("Metrics endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	// Just verify we got some metrics back
	if len(metrics) < 100 {
		t.Errorf("Expected metrics data, got very short response: %d bytes", len(metrics))
	}
}

// TestPresetsEndpoint verifies presets configuration
func TestPresetsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/presets")
	if err != nil {
		t.Fatalf("Presets endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Presets []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			CPUMilli    int    `json:"cpuMilli"`
			MemoryMB    int    `json:"memoryMB"`
			DurationMin int    `json:"durationMin"`
		} `json:"presets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode presets response: %v", err)
	}

	if len(result.Presets) != 3 {
		t.Errorf("Expected 3 presets, got %d", len(result.Presets))
	}
}

// TestProvisionContainer verifies container provisioning flow
func TestProvisionContainer(t *testing.T) {
	payload := fmt.Sprintf(`{
		"imageType": "%s",
		"durationMinutes": %d,
		"cpuMilli": 500,
		"memoryMB": 512
	}`, testImageType, testDuration)

	resp, err := http.Post(
		baseURL+"/api/provision",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("Provision request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID         string    `json:"id"`
		Status     string    `json:"status"`
		ExpiryTime time.Time `json:"expiryTime"`
		CreatedAt  time.Time `json:"createdAt"`
		ImageType  string    `json:"imageType"`
		Cost       float64   `json:"cost"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode provision response: %v", err)
	}

	if result.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", result.Status)
	}

	if result.ImageType != testImageType {
		t.Errorf("Expected imageType '%s', got '%s'", testImageType, result.ImageType)
	}

	// Cleanup
	t.Cleanup(func() {
		deleteContainer(t, result.ID)
	})

	// Wait for async provisioning
	time.Sleep(5 * time.Second)

	// Verify container is running
	containers := listContainers(t)
	found := false
	for _, c := range containers {
		if c.ID == result.ID {
			found = true
			if c.Status != "running" {
				t.Errorf("Expected container status 'running', got '%s'", c.Status)
			}
			break
		}
	}

	if !found {
		t.Error("Provisioned container not found in container list")
	}
}

// TestProvisionContainerWithVolume verifies volume provisioning
func TestProvisionContainerWithVolume(t *testing.T) {
	payload := fmt.Sprintf(`{
		"imageType": "%s",
		"durationMinutes": %d,
		"volumeSizeMB": 512
	}`, testImageType, testDuration)

	resp, err := http.Post(
		baseURL+"/api/provision",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("Provision with volume failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode provision response: %v", err)
	}

	// Cleanup
	t.Cleanup(func() {
		deleteContainer(t, result.ID)
	})

	// Wait for async provisioning
	time.Sleep(5 * time.Second)

	t.Log("Volume provisioning test completed (volume verification requires Docker API)")
}

// TestProvisionValidation verifies input validation
func TestProvisionValidation(t *testing.T) {
	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name:           "missing imageType",
			payload:        `{"durationMinutes": 10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid imageType",
			payload:        `{"imageType": "invalid", "durationMinutes": 10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duration too low",
			payload:        fmt.Sprintf(`{"imageType": "%s", "durationMinutes": 1}`, testImageType),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "volume too large",
			payload:        fmt.Sprintf(`{"imageType": "%s", "durationMinutes": 10, "volumeSizeMB": 99999}`, testImageType),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(
				baseURL+"/api/provision",
				"application/json",
				strings.NewReader(tt.payload),
			)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// Helper: listContainers retrieves all containers
func listContainers(t *testing.T) []struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	ImageType string  `json:"imageType"`
	Cost      float64 `json:"cost"`
} {
	resp, err := http.Get(baseURL + "/api/containers")
	if err != nil {
		t.Fatalf("List containers failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Containers []struct {
			ID        string  `json:"id"`
			Status    string  `json:"status"`
			ImageType string  `json:"imageType"`
			Cost      float64 `json:"cost"`
		} `json:"containers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode containers response: %v", err)
	}

	return result.Containers
}

// Helper: deleteContainer removes a container
func deleteContainer(t *testing.T, containerID string) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/containers/%s", baseURL, containerID), nil)
	if err != nil {
		t.Logf("Failed to create delete request: %v", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Delete request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Delete failed with status %d: %s", resp.StatusCode, string(body))
	}
}
