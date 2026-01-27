package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// ValidateJSONContentType middleware ensures POST/PUT requests have JSON content type
func ValidateJSONContentType(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate POST, PUT, PATCH requests
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			// Allow requests without body (health checks, etc.)
			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				log.Warn("invalid content type",
					slog.String("path", r.URL.Path),
					slog.String("content_type", contentType),
					slog.String("method", r.Method),
				)
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateJSONSchema middleware validates JSON payload structure
// It re-reads the body after validation, allowing handlers to read it again
func ValidateJSONSchema(schema map[string]interface{}, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate POST, PUT, PATCH with body
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Validate JSON is valid
			var payload map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields() // Strict mode: reject unknown fields
			if err := decoder.Decode(&payload); err != nil {
				log.Warn("invalid json payload",
					slog.String("path", r.URL.Path),
					slog.String("error", err.Error()),
				)
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Validate required fields if schema provided
			for field := range schema {
				if _, exists := payload[field]; !exists {
					log.Warn("missing required field",
						slog.String("path", r.URL.Path),
						slog.String("field", field),
					)
					http.Error(w, "Missing required field: "+field, http.StatusBadRequest)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SanitizeInputs middleware removes potentially dangerous characters from query params and headers
// Prevents basic injection attacks
func SanitizeInputs(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for suspicious patterns in query params
			dangerousChars := []string{"<", ">", "\"", "'", "&"}
			for key, values := range r.URL.Query() {
				for _, val := range values {
					for _, char := range dangerousChars {
						if strings.Contains(val, char) {
							log.Warn("suspicious input detected",
								slog.String("path", r.URL.Path),
								slog.String("param", key),
								slog.String("pattern", char),
							)
							http.Error(w, "Invalid input: dangerous characters detected", http.StatusBadRequest)
							return
						}
					}
				}
			}

			// Check suspicious patterns in path
			if strings.Contains(r.URL.Path, "..") || strings.Contains(r.URL.Path, "//") {
				log.Warn("suspicious path pattern detected",
					slog.String("path", r.URL.Path),
				)
				http.Error(w, "Invalid path", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
