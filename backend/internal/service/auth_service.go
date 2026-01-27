package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/containerlease/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo domain.UserRepository
	jwtKey   []byte
	logger   *slog.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo domain.UserRepository,
	jwtKey string,
	logger *slog.Logger,
) *AuthService {
	if logger == nil {
		logger = slog.Default()
	}

	return &AuthService{
		userRepo: userRepo,
		jwtKey:   []byte(jwtKey),
		logger:   logger,
	}
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// RegisterResult represents registration response
type RegisterResult struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// LoginResult represents login response
type LoginResult struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds
	TokenType string `json:"token_type"`
}

// Register creates a new user account
func (s *AuthService) Register(email, username, password, tenantID string) (*RegisterResult, error) {
	// Validate input
	if email == "" || password == "" || username == "" {
		return nil, errors.New("email, username, and password are required")
	}

	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	existingUsername, err := s.userRepo.GetByUsername(username)
	if err == nil && existingUsername != nil {
		return nil, errors.New("username already taken")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", slog.String("error", err.Error()))
		return nil, errors.New("failed to register user")
	}

	// Create user
	user := &domain.User{
		Email:        email,
		Username:     username,
		PasswordHash: string(hash),
		TenantID:     tenantID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("failed to create user", slog.String("error", err.Error()))
		return nil, errors.New("failed to register user")
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Token:    token,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(email, password string) (*LoginResult, error) {
	// Validate input
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	// Get user
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		s.logger.Info("login attempt with non-existent email", slog.String("email", email))
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Info("login failed with wrong password", slog.String("email", email))
		return nil, errors.New("invalid credentials")
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	s.logger.Info("user logged in",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return &LoginResult{
		UserID:    user.ID,
		Email:     user.Email,
		Token:     token,
		ExpiresIn: 900, // 15 minutes
		TokenType: "Bearer",
	}, nil
}

// VerifyToken verifies and parses a JWT token
func (s *AuthService) VerifyToken(tokenString string) (*TokenClaims, error) {
	claims := &TokenClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// generateToken generates a new JWT token for a user
func (s *AuthService) generateToken(user *domain.User) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)

	claims := &TokenClaims{
		UserID:   user.ID,
		Email:    user.Email,
		TenantID: user.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "containerlease",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		s.logger.Error("failed to sign token", slog.String("error", err.Error()))
		return "", errors.New("failed to generate token")
	}

	return tokenString, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	if newPassword == "" || len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash new password", slog.String("error", err.Error()))
		return errors.New("failed to change password")
	}

	// Update user
	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(user); err != nil {
		s.logger.Error("failed to update user password", slog.String("error", err.Error()))
		return errors.New("failed to change password")
	}

	s.logger.Info("user changed password", slog.String("user_id", userID))
	return nil
}
