package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/yourorg/containerlease/internal/security/audit"
	"github.com/yourorg/containerlease/internal/security/auth"
	"github.com/yourorg/containerlease/internal/security/ratelimit"
)

type TenantContextKey struct{}
type ClaimsContextKey struct{}

func JWTMiddleware(tm *auth.TokenManager, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public endpoints
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" ||
				r.URL.Path == "/api/presets" || r.URL.Path == "/api/provision" ||
				r.URL.Path == "/api/containers" || r.URL.Path == "/api/status" ||
				strings.HasPrefix(r.URL.Path, "/api/containers/") ||
				strings.HasPrefix(r.URL.Path, "/ws/logs/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing auth"}`, http.StatusUnauthorized)
				return
			}

			tokenString, err := auth.ExtractToken(authHeader)
			if err != nil {
				http.Error(w, `{"error":"invalid auth"}`, http.StatusUnauthorized)
				return
			}

			claims, err := tm.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsContextKey{}, claims)
			ctx = context.WithValue(ctx, TenantContextKey{}, claims.TenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RateLimitMiddleware(limiter *ratelimit.Limiter, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" ||
				r.URL.Path == "/api/presets" || r.URL.Path == "/api/provision" ||
				r.URL.Path == "/api/containers" || r.URL.Path == "/api/status" ||
				strings.HasPrefix(r.URL.Path, "/api/containers/") ||
				strings.HasPrefix(r.URL.Path, "/ws/logs/") {
				next.ServeHTTP(w, r)
				return
			}

			tenantID := ""
			if t := r.Context().Value(TenantContextKey{}); t != nil {
				tenantID = t.(string)
			}

			if !limiter.Allow(tenantID) {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuditMiddleware(auditLog *audit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := ""
			userID := ""
			if t := r.Context().Value(TenantContextKey{}); t != nil {
				tenantID = t.(string)
			}
			if c := r.Context().Value(ClaimsContextKey{}); c != nil {
				claims := c.(*auth.Claims)
				userID = claims.UserID
			}

			if r.Method == http.MethodPost && r.URL.Path == "/api/provision" {
				auditLog.LogAction(r.Context(), tenantID, userID, "provision", "container", "", "initiated", "")
			}
			if r.Method == http.MethodDelete {
				auditLog.LogAction(r.Context(), tenantID, userID, "delete", "container", r.PathValue("id"), "initiated", "")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetTenantFromContext(ctx context.Context) string {
	if t := ctx.Value(TenantContextKey{}); t != nil {
		return t.(string)
	}
	return ""
}

func GetClaimsFromContext(ctx context.Context) *auth.Claims {
	if c := ctx.Value(ClaimsContextKey{}); c != nil {
		return c.(*auth.Claims)
	}
	return nil
}
