package handler

import (
	"log/slog"
	"net/http"

	"github.com/yourorg/containerlease/internal/service"
)

// DeleteHandler handles container deletion requests
type DeleteHandler struct {
	containerService *service.ContainerService
	logger           *slog.Logger
}

// NewDeleteHandler creates a new delete handler
func NewDeleteHandler(containerService *service.ContainerService, logger *slog.Logger) *DeleteHandler {
	return &DeleteHandler{
		containerService: containerService,
		logger:           logger,
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

	if err := h.containerService.DeleteContainer(r.Context(), containerID); err != nil {
		h.logger.Error("failed to delete container", slog.String("error", err.Error()))
		http.Error(w, "failed to delete container", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
