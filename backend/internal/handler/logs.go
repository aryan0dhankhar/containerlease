package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/aryan0dhankhar/containerlease/internal/domain"
)

// LogsHandler handles WebSocket connections for container logs
type LogsHandler struct {
	dockerClient   domain.DockerClient
	logger         *slog.Logger
	allowedOrigins []string
	containerRepo  domain.ContainerRepository
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(dockerClient domain.DockerClient, logger *slog.Logger, allowedOrigins []string, containerRepo domain.ContainerRepository) *LogsHandler {
	return &LogsHandler{
		dockerClient:   dockerClient,
		logger:         logger,
		allowedOrigins: allowedOrigins,
		containerRepo:  containerRepo,
	}
}

// upgrader is initialized per-request to use instance's allowed origins
func (h *LogsHandler) getUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Allow requests with no origin (e.g., non-browser clients)
				return true
			}
			for _, allowed := range h.allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			h.logger.Warn("websocket origin rejected", slog.String("origin", origin))
			return false
		},
	}
}

// ServeHTTP handles WebSocket requests for container logs
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")

	// Fallback: extract container ID from path manually if PathValue didn't work
	// Path format: /ws/logs/{id}?token=...
	if containerID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			containerID = parts[3]
		}
	}

	h.logger.Debug("logs request", slog.String("container_id", containerID))

	if containerID == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket with origin checking
	upgrader := h.getUpgrader()
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", slog.String("error", err.Error()))
		return
	}
	defer ws.Close()

	// Use request context to avoid premature timeout; allows long-lived streams
	ctx := r.Context()

	// Resolve Docker ID from repository
	container, err := h.containerRepo.GetByID(containerID)
	if err != nil {
		h.logger.Error("container not found for logs", slog.String("container_id", containerID), slog.String("error", err.Error()))
		ws.WriteMessage(websocket.TextMessage, []byte("Error: container not found"))
		return
	}
	if container.DockerID == "" {
		h.logger.Error("container has no docker id", slog.String("container_id", containerID))
		ws.WriteMessage(websocket.TextMessage, []byte("Error: container not yet running"))
		return
	}

	// Get logs from Docker using Docker ID
	logStream, err := h.dockerClient.StreamLogs(ctx, container.DockerID)
	if err != nil {
		h.logger.Error("failed to stream logs",
			slog.String("container_id", containerID),
			slog.String("docker_id", container.DockerID),
			slog.String("error", err.Error()),
		)
		ws.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}
	defer logStream.Close()

	// Stream logs to WebSocket client
	if err := h.streamLogsToWebSocket(ws, logStream, containerID); err != nil {
		h.logger.Debug("log streaming ended",
			slog.String("container_id", containerID),
			slog.String("reason", err.Error()),
		)
	}
}

// streamLogsToWebSocket streams container logs to a WebSocket connection
func (h *LogsHandler) streamLogsToWebSocket(ws *websocket.Conn, logStream io.ReadCloser, containerID string) error {
	scanner := bufio.NewScanner(logStream)
	// Increase the buffer to handle longer log lines safely
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	// Heartbeat ping to keep connection alive
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			case <-done:
				return
			}
		}
	}()
	for scanner.Scan() {
		line := scanner.Bytes()
		if err := ws.WriteMessage(websocket.TextMessage, line); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Debug("websocket closed", slog.String("container_id", containerID))
			}
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		close(done)
		return err
	}
	close(done)
	return nil
}

// GetLogs handles REST API requests for container logs
func (h *LogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	containerID := r.URL.Query().Get("container")
	if containerID == "" {
		// Try path parameter as fallback
		containerID = r.PathValue("id")
	}

	h.logger.Debug("logs REST request", slog.String("container_id", containerID))

	if containerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "missing container id",
		})
		return
	}

	// Resolve Docker ID from repository
	container, err := h.containerRepo.GetByID(containerID)
	if err != nil {
		h.logger.Error("container not found for logs", slog.String("container_id", containerID), slog.String("error", err.Error()))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "container not found",
		})
		return
	}
	if container.DockerID == "" {
		h.logger.Error("container has no docker id", slog.String("container_id", containerID))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "container not yet running",
		})
		return
	}

	// Get logs from Docker using Docker ID
	logStream, err := h.dockerClient.StreamLogs(r.Context(), container.DockerID)
	if err != nil {
		h.logger.Error("failed to fetch logs",
			slog.String("container_id", containerID),
			slog.String("docker_id", container.DockerID),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "failed to fetch logs",
		})
		return
	}
	defer logStream.Close()

	// Read logs in chunks with timeout
	// Instead of waiting for stream to close, read what's available in 2 seconds
	readCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var data []byte
	buffer := make([]byte, 64*1024) // 64KB buffer per read
	maxBytes := 10 * 1024 * 1024    // 10MB total limit

readLoop:
	for {
		// Use a channel to avoid blocking forever on read
		readDone := make(chan int, 1)
		var readErr error

		go func() {
			n, err := logStream.Read(buffer)
			if err != nil && err != io.EOF {
				readErr = err
			}
			readDone <- n
		}()

		select {
		case n := <-readDone:
			if n > 0 {
				data = append(data, buffer[:n]...)
				if len(data) > maxBytes {
					data = data[:maxBytes]
					data = append(data, []byte("\n\n... (logs truncated - exceeded 10MB limit)")...)
					break readLoop
				}
			}
			if readErr != nil || n == 0 {
				// EOF or error, stop reading
				break readLoop
			}
		case <-readCtx.Done():
			// Timeout reached
			if len(data) == 0 {
				data = []byte("(no logs available yet - container may still be initializing)")
			} else {
				data = append(data, []byte("\n\n... (logs incomplete - read timeout)")...)
			}
			break readLoop
		}
	}

	// Return logs as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"logs": string(data),
	})
}
