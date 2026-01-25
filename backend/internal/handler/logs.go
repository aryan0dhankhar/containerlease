package handler

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourorg/containerlease/internal/domain"
)

// LogsHandler handles WebSocket connections for container logs
type LogsHandler struct {
	dockerClient   domain.DockerClient
	logger         *slog.Logger
	allowedOrigins []string
	containerRepo  domain.ContainerRepository
}

// originCtxKey used to pass allowed origins to upgrader
type originCtxKey struct{}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(dockerClient domain.DockerClient, logger *slog.Logger, allowedOrigins []string, containerRepo domain.ContainerRepository) *LogsHandler {
	return &LogsHandler{
		dockerClient:   dockerClient,
		logger:         logger,
		allowedOrigins: allowedOrigins,
		containerRepo:  containerRepo,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		return checkAllowedOrigin(r.Context(), origin)
	},
}

func checkAllowedOrigin(ctx context.Context, origin string) bool {
	allowed := ctx.Value(originCtxKey{})
	if allowedOrigins, ok := allowed.([]string); ok {
		for _, a := range allowedOrigins {
			if a == "*" || a == origin {
				return true
			}
		}
	}
	return false
}

// ServeHTTP handles WebSocket requests for container logs
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	h.logger.Debug("logs request", slog.String("container_id", containerID))

	if containerID == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	// Attach allowed origins to context for upgrader
	r = r.WithContext(context.WithValue(r.Context(), originCtxKey{}, h.allowedOrigins))

	// Upgrade HTTP connection to WebSocket
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
