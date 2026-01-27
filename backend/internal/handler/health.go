package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	dockerClient *client.Client
	redisClient  *redis.Client
	logger       *slog.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	dockerClient *client.Client,
	redisClient *redis.Client,
	logger *slog.Logger,
) *HealthHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &HealthHandler{
		dockerClient: dockerClient,
		redisClient:  redisClient,
		logger:       logger,
	}
}

// HealthResponse represents the health status response
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Health handles GET /health - Simple liveness check
// Returns 200 if the server is running
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

// Ready handles GET /ready - Readiness check for Kubernetes
// Returns 200 only if all dependencies are healthy
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)

	// Check Docker daemon
	dockerOK := false
	if h.dockerClient != nil {
		_, err := h.dockerClient.Ping(ctx)
		if err == nil {
			checks["docker"] = "ok"
			dockerOK = true
		} else {
			checks["docker"] = "error: " + err.Error()
		}
	} else {
		checks["docker"] = "not configured"
	}

	// Check Redis
	redisOK := false
	if h.redisClient != nil {
		err := h.redisClient.Ping(ctx).Err()
		if err == nil {
			checks["redis"] = "ok"
			redisOK = true
		} else {
			checks["redis"] = "error: " + err.Error()
		}
	} else {
		checks["redis"] = "not configured"
	}

	// Determine overall status
	allHealthy := dockerOK && redisOK
	status := "ready"
	statusCode := http.StatusOK

	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status: status,
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("readiness check",
		slog.String("status", status),
		slog.String("docker", checks["docker"]),
		slog.String("redis", checks["redis"]),
	)
}
