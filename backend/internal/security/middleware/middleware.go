package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourorg/containerlease/internal/security/audit"
	"github.com/yourorg/containerlease/internal/security/auth"
	"github.com/yourorg/containerlease/internal/security/ratelimit"
)

type TenantContextKey struct{}
type ClaimsContextKey struct{}

func JWTMiddleware(tm *auth.TokenManager, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for OPTIONS (CORS preflight)
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for public endpoints
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" ||
				r.URL.Path == "/api/login" || r.URL.Path == "/api/presets" ||
				r.URL.Path == "/health" || r.URL.Path == "/ready" ||
				r.URL.Path == "/api/auth/register" || r.URL.Path == "/api/auth/login" {
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
			// Only skip rate limiting for health/metrics endpoints
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Get tenant ID from context if available (for authenticated requests)
			// For unauthenticated requests (like login), use IP address as identifier
			tenantID := ""
			if t := r.Context().Value(TenantContextKey{}); t != nil {
				tenantID = t.(string)
			} else {
				// Use IP address for rate limiting unauthenticated requests
				tenantID = getClientIP(r)
			}

			// Extra strict rate limiting for login endpoint to prevent brute force
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" {
				if !limiter.AllowStrict(tenantID, 10, 5*time.Minute) {
					log.Warn("rate limit exceeded for auth endpoint",
						slog.String("identifier", tenantID),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, `{"error":"too many login attempts, please try again later"}`, http.StatusTooManyRequests)
					return
				}
			} else {
				if !limiter.Allow(tenantID) {
					log.Warn("rate limit exceeded",
						slog.String("identifier", tenantID),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xff {
					if c == ',' {
						return xff[:i]
					}
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
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

// SetTenantInContext sets the tenant ID in the context (useful for testing)
func SetTenantInContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantContextKey{}, tenantID)
}

func GetClaimsFromContext(ctx context.Context) *auth.Claims {
	if c := ctx.Value(ClaimsContextKey{}); c != nil {
		return c.(*auth.Claims)
	}
	return nil
}
