package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/containerlease/internal/service"
	"github.com/yourorg/containerlease/pkg/config"
)

// ProvisionRequest represents the request to provision a container
type ProvisionRequest struct {
	ImageType       string `json:"imageType"`
	DurationMinutes int    `json:"durationMinutes"`
	CPUMilli        int    `json:"cpuMilli,omitempty"`
	MemoryMB        int    `json:"memoryMB,omitempty"`
	LogDemo         bool   `json:"logDemo,omitempty"`
}

// ProvisionResponse represents the response after provisioning
type ProvisionResponse struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	ExpiryTime time.Time `json:"expiryTime"`
	CreatedAt  time.Time `json:"createdAt"`
	ImageType  string    `json:"imageType"`
	Cost       float64   `json:"cost"`
}

// ProvisionHandler handles container provisioning requests
type ProvisionHandler struct {
	containerService *service.ContainerService
	logger           *slog.Logger
	config           *config.Config
}

// NewProvisionHandler creates a new provision handler
func NewProvisionHandler(containerService *service.ContainerService, logger *slog.Logger, cfg *config.Config) *ProvisionHandler {
	return &ProvisionHandler{
		containerService: containerService,
		logger:           logger,
		config:           cfg,
	}
}

// ServeHTTP handles POST /provision requests
func (h *ProvisionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ImageType == "" {
		http.Error(w, "imageType is required", http.StatusBadRequest)
		return
	}

	if !h.isImageAllowed(req.ImageType) {
		http.Error(w, "imageType not allowed", http.StatusBadRequest)
		return
	}

	if req.DurationMinutes < h.config.ContainerMinDuration || req.DurationMinutes > h.config.ContainerMaxDuration {
		http.Error(w, "durationMinutes out of bounds", http.StatusBadRequest)
		return
	}

	// Apply defaults and caps for CPU and memory
	cpuMilli := req.CPUMilli
	if cpuMilli <= 0 {
		cpuMilli = h.config.DefaultCPUMilli
	}
	if cpuMilli > h.config.MaxCPUMilli {
		http.Error(w, "cpuMilli exceeds allowed maximum", http.StatusBadRequest)
		return
	}

	memoryMB := req.MemoryMB
	if memoryMB <= 0 {
		memoryMB = h.config.DefaultMemoryMB
	}
	if memoryMB > h.config.MaxMemoryMB {
		http.Error(w, "memoryMB exceeds allowed maximum", http.StatusBadRequest)
		return
	}

	// Call service layer
	container, err := h.containerService.ProvisionContainer(r.Context(), service.ProvisionOptions{
		ImageType:       req.ImageType,
		DurationMinutes: req.DurationMinutes,
		CPUMilli:        cpuMilli,
		MemoryMB:        memoryMB,
		LogDemo:         req.LogDemo,
	})
	if err != nil {
		h.logger.Error("failed to provision container", slog.String("error", err.Error()))
		http.Error(w, "failed to provision container", http.StatusInternalServerError)
		return
	}

	// Build response
	response := ProvisionResponse{
		ID:         container.ID,
		Status:     container.Status,
		ExpiryTime: container.ExpiryAt,
		CreatedAt:  container.CreatedAt,
		ImageType:  container.ImageType,
		Cost:       container.Cost,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}

func (h *ProvisionHandler) isImageAllowed(image string) bool {
	for _, allowed := range h.config.AllowedImages {
		if allowed == image {
			return true
		}
	}
	return false
}
