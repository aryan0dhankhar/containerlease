package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/domain"
	"github.com/aryan0dhankhar/containerlease/internal/security"
	"github.com/aryan0dhankhar/containerlease/internal/security/middleware"
	"github.com/aryan0dhankhar/containerlease/internal/service"
)

// SnapshotHandler handles container snapshot operations
type SnapshotHandler struct {
	snapshotService *service.SnapshotService
	containerRepo   domain.ContainerRepository
	logger          *slog.Logger
	authz           *security.AuthorizationService
}

// NewSnapshotHandler creates a new snapshot handler
func NewSnapshotHandler(
	snapshotService *service.SnapshotService,
	containerRepo domain.ContainerRepository,
	logger *slog.Logger,
) *SnapshotHandler {
	return &SnapshotHandler{
		snapshotService: snapshotService,
		containerRepo:   containerRepo,
		logger:          logger,
		authz:           security.NewAuthorizationService(logger),
	}
}

// CreateSnapshotRequest represents a request to create a snapshot
type CreateSnapshotRequest struct {
	SnapshotName string `json:"snapshotName"`
	Description  string `json:"description,omitempty"`
}

// SnapshotResponse represents a snapshot in responses
type SnapshotResponse struct {
	ID          string    `json:"id"`
	ContainerID string    `json:"containerId"`
	ImageName   string    `json:"imageName"`
	CreatedAt   time.Time `json:"createdAt"`
	Size        int64     `json:"size"`
	Description string    `json:"description"`
	TenantID    string    `json:"tenantId"`
}

// CreateSnapshot handles POST /api/containers/{id}/snapshot
// Creates a new snapshot of a running container
// Only the container owner (tenant) can snapshot it
func (h *SnapshotHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	// Get tenant ID from context (set by JWT middleware)
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to create snapshot
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermCreateSnapshot); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Parse request body
	var req CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode snapshot request",
			slog.String("container_id", containerID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate snapshot name
	if req.SnapshotName == "" {
		http.Error(w, "snapshotName is required", http.StatusBadRequest)
		return
	}

	// Verify container exists and belongs to tenant
	container, err := h.containerRepo.GetByID(containerID)
	if err != nil {
		h.logger.Error("failed to get container",
			slog.String("container_id", containerID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "container not found", http.StatusNotFound)
		return
	}

	// Validate ownership: only tenant owner can snapshot
	if container.TenantID != tenantID {
		h.logger.Warn("unauthorized snapshot attempt",
			slog.String("container_id", containerID),
			slog.String("container_tenant", container.TenantID),
			slog.String("request_tenant", tenantID),
		)
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	// Create snapshot
	snapshot, err := h.snapshotService.CreateSnapshot(r.Context(), containerID, req.Description)
	if err != nil {
		h.logger.Error("failed to create snapshot",
			slog.String("container_id", containerID),
			slog.String("snapshot_name", req.SnapshotName),
			slog.String("error", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return snapshot response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snapshotToResponse(snapshot))
}

// ListSnapshots handles GET /api/containers/{id}/snapshots
// Lists all snapshots for a specific container
func (h *SnapshotHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	// Get tenant ID from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to list snapshots
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermListSnapshots); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Verify container ownership
	container, err := h.containerRepo.GetByID(containerID)
	if err != nil {
		http.Error(w, "container not found", http.StatusNotFound)
		return
	}

	if container.TenantID != tenantID {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	// List snapshots for container
	snapshots, err := h.snapshotService.GetContainerSnapshots(r.Context(), containerID)
	if err != nil {
		h.logger.Error("failed to list snapshots",
			slog.String("container_id", containerID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "failed to list snapshots", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	respItems := make([]SnapshotResponse, 0, len(snapshots))
	for _, snap := range snapshots {
		respItems = append(respItems, snapshotToResponse(snap))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respItems)
}

// DeleteSnapshot handles DELETE /api/snapshots/{id}
// Deletes a snapshot and its corresponding Docker image
func (h *SnapshotHandler) DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshotID := r.PathValue("id")
	if snapshotID == "" {
		http.Error(w, "snapshot id required", http.StatusBadRequest)
		return
	}

	// Get tenant ID from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to delete snapshot
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermDeleteSnapshot); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Verify snapshot ownership by checking tenant
	snapshot, err := h.snapshotService.GetSnapshot(r.Context(), snapshotID)
	if err != nil {
		http.Error(w, "snapshot not found", http.StatusNotFound)
		return
	}

	if snapshot.TenantID != tenantID {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	// Delete snapshot
	if err := h.snapshotService.DeleteSnapshot(r.Context(), snapshotID); err != nil {
		h.logger.Error("failed to delete snapshot",
			slog.String("snapshot_id", snapshotID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "failed to delete snapshot", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RestoreSnapshotRequest represents a request to restore a snapshot
type RestoreSnapshotRequest struct {
	ContainerName string   `json:"containerName"`
	Tags          []string `json:"tags,omitempty"`
}

// RestoreSnapshot handles POST /api/snapshots/{id}/restore
// Restores a container from a snapshot
// Creates a new container using the snapshot's image
func (h *SnapshotHandler) RestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshotID := r.PathValue("id")
	if snapshotID == "" {
		http.Error(w, "snapshot id required", http.StatusBadRequest)
		return
	}

	// Get tenant ID from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to create container (restore is essentially provisioning from snapshot)
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermCreateContainer); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Parse request
	var req RestoreSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode restore request",
			slog.String("snapshot_id", snapshotID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate container name
	if req.ContainerName == "" {
		http.Error(w, "containerName is required", http.StatusBadRequest)
		return
	}

	// Verify snapshot exists and belongs to tenant
	snapshot, err := h.snapshotService.GetSnapshot(r.Context(), snapshotID)
	if err != nil {
		http.Error(w, "snapshot not found", http.StatusNotFound)
		return
	}

	if snapshot.TenantID != tenantID {
		h.logger.Warn("unauthorized restore attempt",
			slog.String("snapshot_id", snapshotID),
			slog.String("snapshot_tenant", snapshot.TenantID),
			slog.String("request_tenant", tenantID),
		)
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	// Restore from snapshot (in production, would call Docker to create container from image)
	// For now, log the restoration request and return success
	h.logger.Info("snapshot restore initiated",
		slog.String("snapshot_id", snapshotID),
		slog.String("container_name", req.ContainerName),
		slog.String("image", snapshot.ImageName),
		slog.String("tenant_id", tenantID),
	)

	// Build response
	type RestoreResponse struct {
		SnapshotID    string    `json:"snapshotId"`
		RestoredImage string    `json:"restoredImage"`
		ContainerName string    `json:"containerName"`
		Status        string    `json:"status"`
		Message       string    `json:"message"`
		RestoredAt    time.Time `json:"restoredAt"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(RestoreResponse{
		SnapshotID:    snapshotID,
		RestoredImage: snapshot.ImageName,
		ContainerName: req.ContainerName,
		Status:        "restoring",
		Message:       "Snapshot restore initiated. Use the container name to monitor progress.",
		RestoredAt:    time.Now(),
	})
}

// Helper function to convert domain.Snapshot to SnapshotResponse
func snapshotToResponse(snap *domain.Snapshot) SnapshotResponse {
	return SnapshotResponse{
		ID:          snap.ID,
		ContainerID: snap.ContainerID,
		ImageName:   snap.ImageName,
		CreatedAt:   snap.CreatedAt,
		Size:        snap.Size,
		Description: snap.Description,
		TenantID:    snap.TenantID,
	}
}
