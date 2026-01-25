package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/containerlease/internal/handler"
	"github.com/yourorg/containerlease/internal/infrastructure/docker"
	"github.com/yourorg/containerlease/internal/infrastructure/logger"
	"github.com/yourorg/containerlease/internal/infrastructure/redis"
	"github.com/yourorg/containerlease/internal/repository"
	"github.com/yourorg/containerlease/internal/service"
	"github.com/yourorg/containerlease/internal/worker"
	"github.com/yourorg/containerlease/pkg/config"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize structured logger
	log := logger.NewLogger(cfg.LogLevel)
	log.Info("starting ContainerLease server", slog.String("environment", cfg.Environment))

	// 3. Initialize Redis client
	redisClient, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Error("failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	// 4. Initialize Docker client
	dockerClient, err := docker.NewClient(cfg.DockerHost, log)
	if err != nil {
		log.Error("failed to initialize Docker client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5. Initialize repositories
	leaseRepo := repository.NewLeaseRepository(redisClient, log)
	containerRepo := repository.NewContainerRepository(redisClient, log)

	// 6. Initialize services
	containerService := service.NewContainerService(dockerClient, leaseRepo, containerRepo, log, cfg)

	// 7. Initialize handlers
	provisionHandler := handler.NewProvisionHandler(containerService, log, cfg)
	provisionStatusHandler := handler.NewProvisionStatusHandler(containerRepo, log)
	logsHandler := handler.NewLogsHandler(dockerClient, log, cfg.CORSAllowedOrigins, containerRepo)
	statusHandler := handler.NewContainersHandler(containerRepo, log)
	deleteHandler := handler.NewDeleteHandler(containerService, log)

	// 8. Setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("POST /api/provision", provisionHandler)
	mux.Handle("GET /api/containers", statusHandler)
	mux.Handle("GET /api/containers/{id}/status", provisionStatusHandler)
	mux.Handle("DELETE /api/containers/{id}", deleteHandler)
	mux.Handle("GET /ws/logs/{id}", logsHandler)

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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		mux.ServeHTTP(w, r)
	})

	// Health and readiness endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("redis not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Attach request ID middleware for observability
	rootHandler := withRequestID(handlerWithCORS, log)

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

	// 10. Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      rootHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("server starting", slog.Int("port", cfg.ServerPort))

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func generateRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
