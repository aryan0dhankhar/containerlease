package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/security/auth"
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse contains the JWT token
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
}

// LoginHandler handles user authentication
type LoginHandler struct {
	tokenManager *auth.TokenManager
	userStore    *auth.UserStore
	logger       *slog.Logger
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(tm *auth.TokenManager, us *auth.UserStore, logger *slog.Logger) *LoginHandler {
	return &LoginHandler{
		tokenManager: tm,
		userStore:    us,
		logger:       logger,
	}
}

// ServeHTTP handles POST /api/login requests
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode login request", slog.String("error", err.Error()))
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := h.userStore.Authenticate(req.Email, req.Password)
	if err != nil {
		h.logger.Warn("authentication failed",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
		)
		// Generic error to prevent user enumeration
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	expiresIn := 24 * time.Hour
	token, err := h.tokenManager.GenerateToken(user.TenantID, user.ID, user.Email, expiresIn)
	if err != nil {
		h.logger.Error("failed to generate token",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}

	h.logger.Info("user logged in",
		slog.String("user_id", user.ID),
		slog.String("tenant_id", user.TenantID),
		slog.String("email", user.Email),
	)

	response := LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(expiresIn),
		TenantID:  user.TenantID,
		UserID:    user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
