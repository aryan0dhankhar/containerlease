package test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/containerlease/internal/handler"
	"github.com/yourorg/containerlease/internal/infrastructure/logger"
	"github.com/yourorg/containerlease/internal/service"
)

// TestServerHelper creates a test HTTP server without needing a running backend
type TestServerHelper struct {
	Server *httptest.Server
	Logger *slog.Logger
	Mux    *http.ServeMux
}

func NewTestServer(t *testing.T) *TestServerHelper {
	logger := logger.NewLogger("debug")
	mux := http.NewServeMux()

	// Setup basic health endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Setup metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# HELP test_metric Test metric\n# TYPE test_metric counter\n"))
	})

	server := httptest.NewServer(mux)

	return &TestServerHelper{
		Server: server,
		Logger: logger,
		Mux:    mux,
	}
}

func (h *TestServerHelper) Close() {
	h.Server.Close()
}

func (h *TestServerHelper) URL() string {
	return h.Server.URL
}

// AddAuthHandler adds mock auth endpoints to test server
func (h *TestServerHelper) AddAuthHandler(authService *service.AuthService) {
	authHandler := handler.NewAuthHandler(authService, h.Logger)

	h.Mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	h.Mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	h.Mux.HandleFunc("POST /api/auth/change-password", authHandler.ChangePassword)
}

// AssertStatusCode helper function
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	if resp.StatusCode != expected {
		t.Errorf("Expected status %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContentType helper function
func AssertContentType(t *testing.T, resp *http.Response, expected string) {
	if ct := resp.Header.Get("Content-Type"); ct != expected {
		t.Errorf("Expected Content-Type %s, got %s", expected, ct)
	}
}
