package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/internal/security"
	"github.com/yourorg/containerlease/internal/security/middleware"
)

// ContainersHandler handles listing active containers
type ContainersHandler struct {
	containerRepo domain.ContainerRepository
	logger        *slog.Logger
	authz         *security.AuthorizationService
}

// NewContainersHandler creates a new containers handler
func NewContainersHandler(containerRepo domain.ContainerRepository, logger *slog.Logger, authz *security.AuthorizationService) *ContainersHandler {
	return &ContainersHandler{
		containerRepo: containerRepo,
		logger:        logger,
		authz:         authz,
	}
}

// ServeHTTP handles GET /api/containers requests
func (h *ContainersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Debug("containers list request")

	// Get tenant ID from context (set by JWT middleware) and require auth
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: require permission to list containers
	if err := h.authz.ValidatePermission(security.RoleUser, security.PermListContainers); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	containers, err := h.containerRepo.ListByTenant(tenantID)

	if err != nil {
		h.logger.Error("failed to list containers", slog.String("error", err.Error()))
		http.Error(w, "failed to list containers", http.StatusInternalServerError)
		return
	}

	type ContainerResponse struct {
		ID        string  `json:"id"`
		ImageType string  `json:"imageType"`
		Status    string  `json:"status"`
		Cost      float64 `json:"cost"`
		CreatedAt string  `json:"createdAt"`
		ExpiryAt  string  `json:"expiryAt"`
		ExpiresIn int     `json:"expiresIn"`
	}

	respItems := make([]ContainerResponse, 0, len(containers))
	for _, c := range containers {
		remaining := int(time.Until(c.ExpiryAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}

		respItems = append(respItems, ContainerResponse{
			ID:        c.ID,
			ImageType: c.ImageType,
			Status:    c.Status,
			Cost:      c.Cost,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
			ExpiryAt:  c.ExpiryAt.Format(time.RFC3339),
			ExpiresIn: remaining,
		})
	}

	response := map[string]interface{}{
		"containers": respItems,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
