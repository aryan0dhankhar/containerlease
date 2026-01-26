package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret string
	issuer string
}

func NewTokenManager(secret, issuer string) *TokenManager {
	if secret == "" {
		secret = "change-me-in-production"
	}
	if issuer == "" {
		issuer = "containerlease"
	}
	return &TokenManager{secret: secret, issuer: issuer}
}

func (tm *TokenManager) GenerateToken(tenantID, userID, email string, expiresIn time.Duration) (string, error) {
	if tenantID == "" || userID == "" {
		return "", fmt.Errorf("tenant_id and user_id required")
	}
	now := time.Now()
	claims := Claims{
		TenantID: tenantID,
		UserID:   userID,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			Issuer:    tm.issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tm.secret))
}

func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tm.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token failed: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func ExtractToken(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header")
	}
	return parts[1], nil
}
