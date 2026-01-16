package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/containerlease/internal/service"
)

// ProvisionRequest represents the request to provision a container
type ProvisionRequest struct {
	ImageType       string `json:"imageType"`
	DurationMinutes int    `json:"durationMinutes"`
}

// ProvisionResponse represents the response after provisioning
type ProvisionResponse struct {
	ID         string    `json:"id"`
	ExpiryTime time.Time `json:"expiryTime"`
	CreatedAt  time.Time `json:"createdAt"`
	ImageType  string    `json:"imageType"`
}

// ProvisionHandler handles container provisioning requests
type ProvisionHandler struct {
	containerService *service.ContainerService
	logger           *slog.Logger
}

// NewProvisionHandler creates a new provision handler
func NewProvisionHandler(containerService *service.ContainerService, logger *slog.Logger) *ProvisionHandler {
	return &ProvisionHandler{
		containerService: containerService,
		logger:           logger,
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
	if req.ImageType == "" || req.DurationMinutes <= 0 {
		http.Error(w, "invalid imageType or durationMinutes", http.StatusBadRequest)
		return
	}

	// Call service layer
	container, err := h.containerService.ProvisionContainer(r.Context(), req.ImageType, req.DurationMinutes)
	if err != nil {
		h.logger.Error("failed to provision container", slog.String("error", err.Error()))
		http.Error(w, "failed to provision container", http.StatusInternalServerError)
		return
	}

	// Build response
	response := ProvisionResponse{
		ID:         container.ID,
		ExpiryTime: container.ExpiryAt,
		CreatedAt:  container.CreatedAt,
		ImageType:  container.ImageType,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
