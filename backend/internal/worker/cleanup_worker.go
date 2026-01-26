package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/internal/observability/metrics"
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

const archiveRetention = 15 * time.Minute

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
	w.logger.Info("running cleanup check for expired or orphaned containers")

	containers, err := w.containerRepository.List()
	if err != nil {
		w.logger.Error("failed to list containers",
			slog.String("error", err.Error()),
		)
		return
	}
	runningCount := 0
	for _, c := range containers {
		if c.Status == "running" {
			runningCount++
		}
	}
	metrics.SetActive(runningCount)

	now := time.Now()
	for _, c := range containers {
		if now.After(c.ExpiryAt) || now.Equal(c.ExpiryAt) {
			w.cleanupContainer(ctx, c.ID)
			continue
		}

		leaseKey := fmt.Sprintf("lease:%s", c.ID)
		if _, err := w.leaseRepository.GetLease(leaseKey); err != nil {
			w.logger.Info("missing lease for container, cleaning up", slog.String("container_id", c.ID))
			w.cleanupContainer(ctx, c.ID)
		}
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
			metrics.ObserveCleanup("worker", "success")
			return
		}
	}

	// Log final error after all retries
	logger.Error("cleanup failed after retries",
		slog.Int("max_retries", w.maxRetries),
	)
	metrics.ObserveCleanup("worker", "error")
}

// performCleanup executes the actual cleanup steps
func (w *CleanupWorker) performCleanup(ctx context.Context, containerID string) bool {
	logger := w.logger.With(slog.String("container_id", containerID))

	// Get container to find Docker ID
	container, err := w.containerRepository.GetByID(containerID)
	if err != nil {
		logger.Error("failed to get container", slog.String("error", err.Error()))
		return false
	}
	wasRunning := container.Status == "running"

	if container.Status == "terminated" {
		logger.Debug("container already terminated, skipping")
		return true
	}

	// If still pending (no Docker ID), just mark terminated and remove lease
	if container.DockerID == "" {
		logger.Info("container pending without Docker ID, marking terminated")
		container.Status = "terminated"
		container.Cost = calculateCost(container.ImageType, time.Since(container.CreatedAt).Minutes())
		container.ExpiryAt = time.Now().Add(archiveRetention)
		if err := w.containerRepository.Save(container); err != nil {
			logger.Error("failed to persist container", slog.String("error", err.Error()))
			return false
		}
		leaseKey := fmt.Sprintf("lease:%s", containerID)
		if err := w.leaseRepository.DeleteLease(leaseKey); err != nil {
			logger.Error("failed to delete lease", slog.String("error", err.Error()))
			return false
		}
		return true
	}

	// Step 1: Stop Docker container
	if err := w.dockerClient.StopContainer(ctx, container.DockerID); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "no such container") {
			logger.Error("failed to stop container", slog.String("docker_id", container.DockerID), slog.String("error", err.Error()))
			return false
		}
		logger.Warn("container not found during stop, continuing", slog.String("docker_id", container.DockerID))
	} else {
		logger.Debug("container stopped", slog.String("docker_id", container.DockerID))
	}

	// Step 2: Remove Docker container
	if err := w.dockerClient.RemoveContainer(ctx, container.DockerID); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "no such container") {
			logger.Error("failed to remove container", slog.String("docker_id", container.DockerID), slog.String("error", err.Error()))
			return false
		}
		logger.Warn("container not found during remove, continuing", slog.String("docker_id", container.DockerID))
	} else {
		logger.Debug("container removed", slog.String("docker_id", container.DockerID))
	}

	// Step 3: Remove volume if attached
	if container.VolumeID != "" {
		if err := w.dockerClient.RemoveVolume(ctx, container.VolumeID); err != nil {
			logger.Error("failed to remove volume", slog.String("volume_id", container.VolumeID), slog.String("error", err.Error()))
			return false
		}
		logger.Debug("volume removed", slog.String("volume_id", container.VolumeID))
	}

	// Step 4: Mark terminated, compute usage-based cost, retain record briefly
	runtimeMinutes := time.Since(container.CreatedAt).Minutes()
	if runtimeMinutes < 0 {
		runtimeMinutes = 0
	}
	container.Cost = calculateCost(container.ImageType, runtimeMinutes)
	container.Status = "terminated"
	container.ExpiryAt = time.Now().Add(archiveRetention)
	if err := w.containerRepository.Save(container); err != nil {
		logger.Error("failed to persist container", slog.String("error", err.Error()))
		return false
	}

	// Step 5: Delete lease from Redis
	leaseKey := fmt.Sprintf("lease:%s", containerID)
	if err := w.leaseRepository.DeleteLease(leaseKey); err != nil {
		logger.Error("failed to delete lease", slog.String("error", err.Error()))
		return false
	}
	logger.Debug("deleted lease")

	if wasRunning {
		metrics.DecrementActive()
	}

	return true
}

func calculateCost(imageType string, durationMinutes float64) float64 {
	hourlyRate := 0.0
	switch imageType {
	case "ubuntu":
		hourlyRate = 0.04
	case "alpine":
		hourlyRate = 0.01
	default:
		hourlyRate = 0.04
	}
	if durationMinutes < 0 {
		durationMinutes = 0
	}
	return hourlyRate * (durationMinutes / 60.0)
}
