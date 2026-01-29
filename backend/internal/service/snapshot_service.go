package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/domain"
	"github.com/aryan0dhankhar/containerlease/pkg/config"
)

// SnapshotService handles container snapshot/backup operations (Phase 2: Disaster Recovery)
type SnapshotService struct {
	dockerClient        domain.DockerClient
	containerRepository domain.ContainerRepository
	snapshotRepository  domain.SnapshotRepository
	logger              *slog.Logger
	config              *config.Config
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(
	dockerClient domain.DockerClient,
	containerRepo domain.ContainerRepository,
	snapshotRepo domain.SnapshotRepository,
	logger *slog.Logger,
	cfg *config.Config,
) *SnapshotService {
	return &SnapshotService{
		dockerClient:        dockerClient,
		containerRepository: containerRepo,
		snapshotRepository:  snapshotRepo,
		logger:              logger,
		config:              cfg,
	}
}

// CreateSnapshot saves a running container's state as a snapshot
// This creates a Docker image from the container
func (s *SnapshotService) CreateSnapshot(ctx context.Context, containerID string, description string) (*domain.Snapshot, error) {
	// Get container
	container, err := s.containerRepository.GetByID(containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	if container.Status != "running" {
		return nil, fmt.Errorf("can only snapshot running containers, current status: %s", container.Status)
	}

	if container.DockerID == "" {
		return nil, fmt.Errorf("container has no Docker ID, cannot snapshot")
	}

	// Create snapshot record
	snapshot := &domain.Snapshot{
		ID:          generateSnapshotID(),
		ContainerID: containerID,
		ImageName:   fmt.Sprintf("snapshot-%s-%d", containerID, time.Now().Unix()),
		CreatedAt:   time.Now(),
		Description: description,
		TenantID:    container.TenantID,
	}

	logger := s.logger.With(
		slog.String("container_id", containerID),
		slog.String("snapshot_id", snapshot.ID),
		slog.String("image_name", snapshot.ImageName),
	)

	logger.Info("creating snapshot from running container")

	// Commit Docker container to image
	if err := s.dockerClient.CommitContainer(ctx, container.DockerID, snapshot.ImageName); err != nil {
		logger.Error("failed to commit container to image",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to commit container: %w", err)
	}

	logger.Debug("container committed to image")

	// Save snapshot metadata
	if err := s.snapshotRepository.Create(snapshot); err != nil {
		logger.Error("failed to save snapshot metadata",
			slog.String("error", err.Error()),
		)
		// Clean up the image since we failed to save metadata
		_ = s.dockerClient.RemoveImage(ctx, snapshot.ImageName)
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	logger.Info("snapshot created successfully", slog.String("snapshot_id", snapshot.ID))
	return snapshot, nil
}

// RestoreSnapshot creates a new container from a snapshot
func (s *SnapshotService) RestoreSnapshot(ctx context.Context, snapshotID string, opts ProvisionOptions) (*domain.Container, error) {
	// Get snapshot
	snapshot, err := s.snapshotRepository.GetByID(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	logger := s.logger.With(
		slog.String("snapshot_id", snapshotID),
		slog.String("snapshot_image", snapshot.ImageName),
	)

	logger.Info("restoring container from snapshot")

	// Create new container entity
	now := time.Now()
	expiryTime := now.Add(time.Duration(opts.DurationMinutes) * time.Minute)

	container := &domain.Container{
		ID:          generateContainerID(),
		TenantID:    snapshot.TenantID,
		ImageType:   opts.ImageType,
		CPUMilli:    opts.CPUMilli,
		MemoryMB:    opts.MemoryMB,
		Status:      "pending",
		CreatedAt:   now,
		ExpiryAt:    expiryTime,
		MaxRestarts: 3, // Phase 2: Self-healing
	}

	// Save container - Note: This is a simplified restore
	// In production, you'd need a LeaseRepository in SnapshotService
	// or better yet, delegate to ContainerService.ProvisionContainer
	if err := s.containerRepository.Save(container); err != nil {
		return nil, fmt.Errorf("failed to save container: %w", err)
	}

	logger.Info("container restored from snapshot",
		slog.String("container_id", container.ID),
	)

	return container, nil
}

// ListSnapshots returns all snapshots for a tenant
func (s *SnapshotService) ListSnapshots(ctx context.Context, tenantID string) ([]*domain.Snapshot, error) {
	snapshots, err := s.snapshotRepository.GetByTenant(tenantID)
	if err != nil {
		s.logger.Error("failed to list snapshots",
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	return snapshots, nil
}

// GetContainerSnapshots returns all snapshots for a specific container
func (s *SnapshotService) GetContainerSnapshots(ctx context.Context, containerID string) ([]*domain.Snapshot, error) {
	snapshots, err := s.snapshotRepository.GetByContainerID(containerID)
	if err != nil {
		s.logger.Error("failed to get container snapshots",
			slog.String("container_id", containerID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	return snapshots, nil
}

// GetSnapshot retrieves a snapshot by ID
func (s *SnapshotService) GetSnapshot(ctx context.Context, snapshotID string) (*domain.Snapshot, error) {
	snapshot, err := s.snapshotRepository.GetByID(snapshotID)
	if err != nil {
		s.logger.Error("failed to get snapshot",
			slog.String("snapshot_id", snapshotID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}
	return snapshot, nil
}

// DeleteSnapshot removes a snapshot
func (s *SnapshotService) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	snapshot, err := s.snapshotRepository.GetByID(snapshotID)
	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}

	logger := s.logger.With(
		slog.String("snapshot_id", snapshotID),
		slog.String("image_name", snapshot.ImageName),
	)

	// Remove Docker image
	if err := s.dockerClient.RemoveImage(ctx, snapshot.ImageName); err != nil {
		logger.Warn("failed to remove Docker image",
			slog.String("error", err.Error()),
		)
		// Continue with deleting metadata anyway
	}

	// Remove snapshot metadata
	if err := s.snapshotRepository.Delete(snapshotID); err != nil {
		logger.Error("failed to delete snapshot metadata",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	logger.Info("snapshot deleted successfully")
	return nil
}

// generateSnapshotID generates a unique snapshot ID
func generateSnapshotID() string {
	return fmt.Sprintf("snapshot-%d", time.Now().UnixNano())
}
