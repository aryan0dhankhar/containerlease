package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
)

// ProvisionStatusResponse represents the current status of a provisioning container
type ProvisionStatusResponse struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"` // pending, running, error
	ImageType  string    `json:"imageType"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiryTime time.Time `json:"expiryTime"`
	Cost       float64   `json:"cost"`
	Error      string    `json:"error,omitempty"`
	TimeLeft   int       `json:"timeLeftSeconds"` // Seconds remaining
}

// ProvisionStatusHandler handles GET /api/containers/{id} requests for real-time status
type ProvisionStatusHandler struct {
	containerRepo domain.ContainerRepository
	logger        *slog.Logger
}

// NewProvisionStatusHandler creates a new provision status handler
func NewProvisionStatusHandler(containerRepo domain.ContainerRepository, logger *slog.Logger) *ProvisionStatusHandler {
	return &ProvisionStatusHandler{
		containerRepo: containerRepo,
		logger:        logger,
	}
}

// ServeHTTP handles GET /api/containers/{id}/status requests
func (h *ProvisionStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	container, err := h.containerRepo.GetByID(containerID)
	if err != nil {
		h.logger.Error("failed to get container", slog.String("container_id", containerID), slog.String("error", err.Error()))
		http.Error(w, "container not found", http.StatusNotFound)
		return
	}

	// Calculate time remaining
	now := time.Now()
	timeLeft := int(container.ExpiryAt.Sub(now).Seconds())
	if timeLeft < 0 {
		timeLeft = 0
	}

	response := ProvisionStatusResponse{
		ID:         container.ID,
		Status:     container.Status,
		ImageType:  container.ImageType,
		CreatedAt:  container.CreatedAt,
		ExpiryTime: container.ExpiryAt,
		Cost:       container.Cost,
		Error:      container.Error,
		TimeLeft:   timeLeft,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
