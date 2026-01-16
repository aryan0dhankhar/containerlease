package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
)

// CleanupWorker periodically checks for expired container leases and cleans them up
// This is the CORE OF THE CLEANUP LOGIC
type CleanupWorker struct {
	leaseRepository     domain.LeaseRepository
	containerRepository domain.ContainerRepository
	dockerClient        domain.DockerClient
	logger              *slog.Logger
	interval            time.Duration
	maxRetries          int
}

// NewCleanupWorker creates a new cleanup worker
func NewCleanupWorker(
	leaseRepo domain.LeaseRepository,
	containerRepo domain.ContainerRepository,
	dockerClient domain.DockerClient,
	logger *slog.Logger,
	interval time.Duration,
) *CleanupWorker {
	return &CleanupWorker{
		leaseRepository:     leaseRepo,
		containerRepository: containerRepo,
		dockerClient:        dockerClient,
		logger:              logger,
		interval:            interval,
		maxRetries:          3,
	}
}

// Start begins the cleanup worker loop
// This runs continuously in a goroutine checking for expired leases
func (w *CleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("cleanup worker started", slog.Duration("interval", w.interval))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("cleanup worker stopped")
			return
		case <-ticker.C:
			w.cleanupExpiredContainers(ctx)
		}
	}
}

// cleanupExpiredContainers is the main cleanup routine
func (w *CleanupWorker) cleanupExpiredContainers(ctx context.Context) {
	// Query Redis for all expired leases
	expiredLeases, err := w.leaseRepository.GetExpiredLeases()
	if err != nil {
		w.logger.Error("failed to get expired leases",
			slog.String("error", err.Error()),
		)
		return
	}

	if len(expiredLeases) == 0 {
		// No expired containers
		return
	}

	w.logger.Info("found expired leases", slog.Int("count", len(expiredLeases)))

	// Clean up each expired container
	for _, containerID := range expiredLeases {
		w.cleanupContainer(ctx, containerID)
	}
}

// cleanupContainer removes a single container with retry logic
func (w *CleanupWorker) cleanupContainer(ctx context.Context, containerID string) {
	logger := w.logger.With(slog.String("container_id", containerID))

	// Retry logic with exponential backoff
	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(attempt*attempt) * time.Second
			logger.Warn("retrying cleanup", slog.Int("attempt", attempt), slog.Duration("backoff", backoff))
			time.Sleep(backoff)
		}

		if w.performCleanup(ctx, containerID) {
			logger.Info("cleanup successful")
			return
		}
	}

	// Log final error after all retries
	logger.Error("cleanup failed after retries",
		slog.Int("max_retries", w.maxRetries),
	)
}

// performCleanup executes the actual cleanup steps
func (w *CleanupWorker) performCleanup(ctx context.Context, containerID string) bool {
	logger := w.logger.With(slog.String("container_id", containerID))

	// Step 1: Stop Docker container
	if err := w.dockerClient.StopContainer(ctx, containerID); err != nil {
		logger.Error("failed to stop container", slog.String("error", err.Error()))
		return false
	}
	logger.Debug("container stopped")

	// Step 2: Remove Docker container
	if err := w.dockerClient.RemoveContainer(ctx, containerID); err != nil {
		logger.Error("failed to remove container", slog.String("error", err.Error()))
		return false
	}
	logger.Debug("container removed")

	// Step 3: Delete from container repository
	if err := w.containerRepository.Delete(containerID); err != nil {
		logger.Error("failed to delete from container repository", slog.String("error", err.Error()))
		return false
	}
	logger.Debug("deleted from container repository")

	// Step 4: Delete lease from Redis
	leaseKey := fmt.Sprintf("lease:%s", containerID)
	if err := w.leaseRepository.DeleteLease(leaseKey); err != nil {
		logger.Error("failed to delete lease", slog.String("error", err.Error()))
		return false
	}
	logger.Debug("deleted lease")

	return true
}
