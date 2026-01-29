package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/handler"
	"github.com/aryan0dhankhar/containerlease/internal/infrastructure/docker"
	"github.com/aryan0dhankhar/containerlease/internal/infrastructure/logger"
	"github.com/aryan0dhankhar/containerlease/internal/infrastructure/redis"
	obsmetrics "github.com/aryan0dhankhar/containerlease/internal/observability/metrics"
	"github.com/aryan0dhankhar/containerlease/internal/observability/tracing"
	"github.com/aryan0dhankhar/containerlease/internal/repository"
	"github.com/aryan0dhankhar/containerlease/internal/security"
	"github.com/aryan0dhankhar/containerlease/internal/security/audit"
	"github.com/aryan0dhankhar/containerlease/internal/security/auth"
	"github.com/aryan0dhankhar/containerlease/internal/security/middleware"
	"github.com/aryan0dhankhar/containerlease/internal/security/ratelimit"
	"github.com/aryan0dhankhar/containerlease/internal/service"
	"github.com/aryan0dhankhar/containerlease/internal/worker"
	"github.com/aryan0dhankhar/containerlease/pkg/config"
	"github.com/aryan0dhankhar/containerlease/pkg/database"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// 0. Validate required environment variables
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret == "" {
		fmt.Fprintf(os.Stderr, "FATAL: JWT_SECRET environment variable is required\n")
		os.Exit(1)
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost == "" && os.Getenv("ENVIRONMENT") == "production" {
		fmt.Fprintf(os.Stderr, "FATAL: DB_HOST environment variable required in production\n")
		os.Exit(1)
	}

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize structured logger
	log := logger.NewLogger(cfg.LogLevel)
	log.Info("starting ContainerLease server", slog.String("environment", cfg.Environment))

	// 2a. Initialize tracing (no-op if endpoint not set)
	shutdownTracing, err := tracing.Init(context.Background(), log, "containerlease", cfg.Environment)
	if err != nil {
		log.Error("failed to initialize tracing", slog.String("error", err.Error()))
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	// 3. Initialize Redis client (optional - allows running without Redis)
	var redisClient *redis.Client
	if cfg.RedisURL != "" {
		var err error
		redisClient, err = redis.NewClient(cfg.RedisURL)
		if err != nil {
			log.Warn("failed to connect to Redis - running without cache", slog.String("error", err.Error()))
			redisClient = nil
		}
	} else {
		log.Info("Redis URL not configured - running without cache")
	}

	// Ensure cleanup on exit
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 4. Initialize Docker client
	dockerClient, err := docker.NewClient(cfg.DockerHost, log)
	if err != nil {
		log.Error("failed to initialize Docker client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5. Initialize repositories (Redis-backed for containers/leases)
	leaseRepo := repository.NewLeaseRepository(redisClient, log)
	containerRepo := repository.NewContainerRepository(redisClient, log)

	// 5a. Initialize PostgreSQL connection (for users/tenants/auth)
	dbCfg := database.DefaultConfig()
	// Override with env vars if set
	if h := os.Getenv("DB_HOST"); h != "" {
		dbCfg.Host = h
	}
	if u := os.Getenv("DB_USER"); u != "" {
		dbCfg.User = u
	}
	if p := os.Getenv("DB_PASSWORD"); p != "" {
		dbCfg.Password = p
	}
	if d := os.Getenv("DB_NAME"); d != "" {
		dbCfg.Database = d
	}
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()
	dbPool, err := database.NewConnectionPool(dbCtx, dbCfg, log)
	if err != nil {
		log.Error("failed to connect to Postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	// 5b. Run database migrations
	if err := runMigrations(dbCtx, dbPool.GetDB(), log); err != nil {
		log.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5c. SQL-backed repositories
	userRepo := repository.NewPostgresUserRepository(dbPool.GetDB(), log)
	_ = userRepo // used by auth service

	// 6. Initialize services
	containerService := service.NewContainerService(dockerClient, leaseRepo, containerRepo, log, cfg)
	authService := service.NewAuthService(userRepo, os.Getenv("JWT_SECRET"), log)

	// 7. Initialize security components
	tokenManager := auth.NewTokenManager(os.Getenv("JWT_SECRET"), "containerlease")
	userStore := auth.NewUserStore()
	rateLimiter := ratelimit.NewLimiter(100, time.Minute) // 100 requests per minute per tenant
	auditLogger := audit.NewLogger(log)
	authz := security.NewAuthorizationService(log)

	// 7a. Initialize handlers
	loginHandler := handler.NewLoginHandler(tokenManager, userStore, log)
	// New auth endpoints backed by Postgres users
	authHandler := handler.NewAuthHandler(authService, log)
	provisionHandler := handler.NewProvisionHandler(containerService, log, cfg, authz)
	provisionStatusHandler := handler.NewProvisionStatusHandler(containerRepo, log)
	presetsHandler := handler.NewPresetsHandler(cfg, log)
	logsHandler := handler.NewLogsHandler(dockerClient, log, cfg.CORSAllowedOrigins, containerRepo)
	statusHandler := handler.NewContainersHandler(containerRepo, log, authz)
	deleteHandler := handler.NewDeleteHandler(containerService, log, authz)

	// 8. Setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("POST /api/login", loginHandler)
	// New auth routes
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/change-password", authHandler.ChangePassword)
	mux.Handle("POST /api/provision", provisionHandler)
	mux.Handle("GET /api/presets", presetsHandler)
	mux.Handle("GET /api/containers", statusHandler)
	mux.Handle("GET /api/containers/{id}/status", provisionStatusHandler)
	mux.Handle("DELETE /api/containers/{id}", deleteHandler)
	mux.Handle("GET /api/logs", http.HandlerFunc(logsHandler.GetLogs))
	// WebSocket logs endpoint - handled separately without OpenTelemetry wrapping
	mux.Handle("GET /ws/logs/{id}", logsHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// CORS middleware honoring configured origins
	handlerWithCORS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originAllowed(cfg.CORSAllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(cfg.CORSAllowedOrigins) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", cfg.CORSAllowedOrigins[0])
		}
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		mux.ServeHTTP(w, r)
	})

	// Health and readiness endpoints (no auth required)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check Redis if available
		if redisClient != nil {
			if err := redisClient.Ping(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("redis not ready"))
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Main handler with all middleware layers for regular HTTP endpoints
	// Base app handler with CORS
	// Order matters: rate limit protects all endpoints, JWT validates protected ones
	base := withRequestID(
		middleware.RateLimitMiddleware(rateLimiter, log)(
			middleware.JWTMiddleware(tokenManager, log)(
				middleware.AuditMiddleware(auditLogger)(handlerWithCORS),
			),
		),
		log,
	)

	// Add HTTP metrics middleware
	withMetrics := obsmetrics.HTTPMetricsMiddleware(base)

	// Wrap with OpenTelemetry HTTP handler for tracing
	rootHandler := otelhttp.NewHandler(withMetrics, "http.server")

	// 9. Start cleanup worker in background
	cleanupWorker := worker.NewCleanupWorker(
		leaseRepo,
		containerRepo,
		dockerClient,
		log,
		time.Duration(cfg.CleanupIntervalMinutes)*time.Minute,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cleanupWorker.Start(ctx)

	// Combined handler: WebSocket routes bypass middleware wrapping, other routes go through full middleware stack
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route WebSocket connections directly to the logs handler without heavy middleware
		if r.Method == http.MethodGet && len(r.URL.Path) > 8 && r.URL.Path[:8] == "/ws/logs" {
			log.Debug("websocket handler intercepted", slog.String("path", r.URL.Path))
			// Apply CORS headers manually
			origin := r.Header.Get("Origin")
			if originAllowed(cfg.CORSAllowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(cfg.CORSAllowedOrigins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", cfg.CORSAllowedOrigins[0])
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")

			// Apply JWT middleware for WebSocket
			authHeader := r.Header.Get("Authorization")
			token := r.URL.Query().Get("token")

			if token == "" && authHeader != "" {
				var err error
				token, err = auth.ExtractToken(authHeader)
				if err != nil {
					http.Error(w, `{"error":"invalid auth"}`, http.StatusUnauthorized)
					return
				}
			}

			if token == "" {
				http.Error(w, `{"error":"missing auth"}`, http.StatusUnauthorized)
				return
			}

			claims, err := tokenManager.ValidateToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), middleware.ClaimsContextKey{}, claims)
			ctx = context.WithValue(ctx, middleware.TenantContextKey{}, claims.TenantID)

			// Extract container ID from path manually (path format: /ws/logs/{id})
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 4 {
				containerID := parts[3]
				// Create a new request with the container ID in URL values for PathValue compatibility
				r = r.WithContext(context.WithValue(ctx, "container_id", containerID))
			}

			logsHandler.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Route all other requests through the full middleware stack
		rootHandler.ServeHTTP(w, r)
	})

	// 10. Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("server starting",
		slog.Int("port", cfg.ServerPort),
		slog.String("auth", "jwt"),
		slog.Int("rate_limit", 100),
		slog.String("rate_limit_window", "1m"),
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// TLS support: if cert/key files are set via env, use HTTPS
		certFile := os.Getenv("TLS_CERT_FILE")
		keyFile := os.Getenv("TLS_KEY_FILE")

		var err error
		if certFile != "" && keyFile != "" {
			log.Info("TLS enabled", slog.String("cert", certFile), slog.String("key", keyFile))
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Warn("TLS not configured - running in HTTP mode (insecure for production)")
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("shutdown signal received")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", slog.String("error", err.Error()))
	}

	cancel() // Stop cleanup worker
	rateLimiter.Stop()
	log.Info("server stopped")
}

type requestIDKey struct{}

// withRequestID attaches a request ID to the context and response headers for traceability
func withRequestID(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := generateRequestID()
		w.Header().Set("X-Request-ID", reqID)

		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		start := time.Now()

		next.ServeHTTP(w, r.WithContext(ctx))

		log.Info("request completed",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration_ms", time.Since(start)),
		)
	})
}

func generateRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func runMigrations(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	// Simple migration runner: read SQL files from migrations dir and execute
	migrationsDir := "migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no migrations directory found")
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", path, err)
		}

		// Execute migration (idempotent: ignore "already exists" errors)
		_, err = db.ExecContext(ctx, string(sqlBytes))
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate") {
				log.Warn("migration execution warning", slog.String("file", entry.Name()), slog.String("error", err.Error()))
			}
		} else {
			log.Info("migration applied", slog.String("file", entry.Name()))
		}
	}
	return nil
}

func originAllowed(allowed []string, origin string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
