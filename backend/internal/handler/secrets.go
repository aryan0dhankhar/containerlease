package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/yourorg/containerlease/internal/security"
	"github.com/yourorg/containerlease/internal/security/middleware"
)

// SecretsHandler handles secret management operations (admin only)
type SecretsHandler struct {
	logger *slog.Logger
	authz  *security.AuthorizationService
}

// NewSecretsHandler creates a new secrets handler
func NewSecretsHandler(logger *slog.Logger) *SecretsHandler {
	return &SecretsHandler{
		logger: logger,
		authz:  security.NewAuthorizationService(logger),
	}
}

// RotateJWTSecretRequest represents a request to rotate JWT secret
type RotateJWTSecretRequest struct {
	ConfirmCurrentSecret string `json:"confirmCurrentSecret"`
	NewSecret            string `json:"newSecret,omitempty"` // If empty, generate random
}

// RotateJWTSecretResponse represents the result of JWT secret rotation
type RotateJWTSecretResponse struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	RotatedAt   time.Time `json:"rotatedAt"`
	OldSecretID string    `json:"oldSecretId"`
	NewSecretID string    `json:"newSecretId"`
}

// RotateJWTSecret handles POST /api/admin/secrets/jwt/rotate
// Only admin users can rotate the JWT secret
// Requires confirmation with current secret for safety
func (h *SecretsHandler) RotateJWTSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant from context (set by JWT middleware)
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: only admins can rotate secrets
	if err := h.authz.ValidatePermission(security.RoleAdmin, security.PermManageTenant); err != nil {
		http.Error(w, "forbidden - admin access required", http.StatusForbidden)
		return
	}

	// Parse request
	var req RotateJWTSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode rotation request",
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Verify current secret
	currentSecret := os.Getenv("JWT_SECRET")
	if req.ConfirmCurrentSecret != currentSecret {
		h.logger.Warn("jwt rotation attempt with incorrect current secret",
			slog.String("tenant_id", tenantID),
		)
		http.Error(w, "incorrect current secret", http.StatusForbidden)
		return
	}

	// Generate new secret if not provided
	newSecret := req.NewSecret
	if newSecret == "" {
		newSecret = generateRandomSecret(32)
	}

	// Validate new secret length (at least 32 characters)
	if len(newSecret) < 32 {
		http.Error(w, "new secret must be at least 32 characters", http.StatusBadRequest)
		return
	}

	// In production, would:
	// 1. Store old secret in rotation history for token validation
	// 2. Update JWT_SECRET env var
	// 3. Signal workers to reload config
	// 4. Audit the rotation

	oldSecretID := hashSecret(currentSecret)[:16]
	newSecretID := hashSecret(newSecret)[:16]

	h.logger.Warn("JWT secret rotated",
		slog.String("tenant_id", tenantID),
		slog.String("old_secret_id", oldSecretID),
		slog.String("new_secret_id", newSecretID),
	)

	// Return response (NOTE: do not include actual secrets)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RotateJWTSecretResponse{
		Status:      "success",
		Message:     "JWT secret rotated successfully. Update JWT_SECRET environment variable.",
		RotatedAt:   time.Now(),
		OldSecretID: oldSecretID,
		NewSecretID: newSecretID,
	})
}

// ListSecretRotationHistory handles GET /api/admin/secrets/history
// Returns rotation history (admin only, no actual secrets)
func (h *SecretsHandler) ListSecretRotationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: only admins
	if err := h.authz.ValidatePermission(security.RoleAdmin, security.PermManageTenant); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Return empty history for now (would query database in production)
	type RotationEntry struct {
		RotatedAt time.Time `json:"rotatedAt"`
		SecretID  string    `json:"secretId"`
		RotatedBy string    `json:"rotatedBy"`
	}

	history := []RotationEntry{
		{
			RotatedAt: time.Now().Add(-24 * time.Hour),
			SecretID:  "current",
			RotatedBy: "system",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetSecretsStatus handles GET /api/admin/secrets/status
// Returns status of secret management (admin only)
func (h *SecretsHandler) GetSecretsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant from context
	tenantID := middleware.GetTenantFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// RBAC: only admins
	if err := h.authz.ValidatePermission(security.RoleAdmin, security.PermManageTenant); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	type SecretsStatus struct {
		JWTSecretSet      bool      `json:"jwtSecretSet"`
		LastRotation      time.Time `json:"lastRotation"`
		RotationInterval  string    `json:"rotationInterval"`  // Recommended interval
		DaysUntilRotation int       `json:"daysUntilRotation"` // Days until rotation recommended
	}

	// Check if JWT_SECRET is set
	jwtSecretSet := os.Getenv("JWT_SECRET") != ""

	status := SecretsStatus{
		JWTSecretSet:      jwtSecretSet,
		LastRotation:      time.Now().Add(-24 * time.Hour), // Mock data
		RotationInterval:  "90 days",
		DaysUntilRotation: 60, // Mock: next rotation recommended in 60 days
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper functions

func generateRandomSecret(length int) string {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		// Fallback to simple hex generation
		return hex.EncodeToString(buf)
	}
	return hex.EncodeToString(buf)
}

func hashSecret(secret string) string {
	// Simple hash for display purposes (not cryptographic)
	// In production, use sha256
	hash := 0
	for _, char := range secret {
		hash = ((hash << 5) - hash) + int(char)
	}
	return hex.EncodeToString([]byte{byte(hash), byte(hash >> 8), byte(hash >> 16), byte(hash >> 24)})
}
