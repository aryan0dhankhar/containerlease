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
	dockerClient domain.DockerClient
	logger       *slog.Logger
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(dockerClient domain.DockerClient, logger *slog.Logger) *LogsHandler {
	return &LogsHandler{
		dockerClient: dockerClient,
		logger:       logger,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev; restrict in production
	},
}

// ServeHTTP handles WebSocket requests for container logs
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	h.logger.Debug("logs request", slog.String("container_id", containerID))

	if containerID == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", slog.String("error", err.Error()))
		return
	}
	defer ws.Close()

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get logs from Docker
	logStream, err := h.dockerClient.StreamLogs(ctx, containerID)
	if err != nil {
		h.logger.Error("failed to stream logs",
			slog.String("container_id", containerID),
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
		return err
	}

	return nil
}
