package handler

import (
	"log/slog"
	"net/http"

	"github.com/yourorg/containerlease/internal/security"
	"github.com/yourorg/containerlease/internal/security/middleware"
	"github.com/yourorg/containerlease/internal/service"
)

// DeleteHandler handles container deletion requests
type DeleteHandler struct {
	containerService *service.ContainerService
	logger           *slog.Logger
	authz            *security.AuthorizationService
}

// NewDeleteHandler creates a new delete handler
func NewDeleteHandler(containerService *service.ContainerService, logger *slog.Logger, authz *security.AuthorizationService) *DeleteHandler {
	return &DeleteHandler{
		containerService: containerService,
		logger:           logger,
		authz:            authz,
	}
}

// ServeHTTP handles DELETE /api/containers/{id} requests
func (h *DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("delete container request", slog.String("container_id", containerID))

	// Get tenant ID from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		h.logger.Error("tenant ID not found in context")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to delete containers
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermDeleteContainer); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Verify tenant owns this container before deleting
	container, err := h.containerService.GetContainer(r.Context(), containerID)
	if err != nil {
		h.logger.Error("failed to get container for ownership check", slog.String("error", err.Error()))
		http.Error(w, "container not found", http.StatusNotFound)
		return
	}

	if container.TenantID != tenantID {
		h.logger.Warn("tenant attempted to delete another tenant's container",
			slog.String("tenant_id", tenantID),
			slog.String("container_tenant", container.TenantID),
			slog.String("container_id", containerID),
		)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.containerService.DeleteContainer(r.Context(), containerID); err != nil {
		h.logger.Error("failed to delete container", slog.String("error", err.Error()))
		http.Error(w, "failed to delete container", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
