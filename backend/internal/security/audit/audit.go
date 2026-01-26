package audit

import (
	"context"
	"log/slog"
	"time"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{logger: logger}
}

func (al *Logger) LogAction(ctx context.Context, tenantID, userID, action, resource, resourceID, status, details string) {
	requestID := ""
	if reqID := ctx.Value("request_id"); reqID != nil {
		requestID = reqID.(string)
	}

	al.logger.Info("audit",
		slog.String("action", action),
		slog.String("resource", resource),
		slog.String("resource_id", resourceID),
		slog.String("tenant_id", tenantID),
		slog.String("user_id", userID),
		slog.String("status", status),
		slog.String("details", details),
		slog.String("request_id", requestID),
		slog.Time("timestamp", time.Now()),
	)
}

func (al *Logger) LogProvisioning(ctx context.Context, tenantID, userID, containerID, status, details string) {
	al.LogAction(ctx, tenantID, userID, "provision", "container", containerID, status, details)
}

func (al *Logger) LogDeletion(ctx context.Context, tenantID, userID, containerID, status, details string) {
	al.LogAction(ctx, tenantID, userID, "delete", "container", containerID, status, details)
}

func (al *Logger) LogDenied(ctx context.Context, tenantID, userID, reason string) {
	al.LogAction(ctx, tenantID, userID, "access_denied", "api", "", "denied", reason)
}
