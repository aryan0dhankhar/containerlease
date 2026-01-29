package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/domain"
	"github.com/aryan0dhankhar/containerlease/internal/observability/metrics"
	"github.com/aryan0dhankhar/containerlease/pkg/config"
)

// ContainerService handles container provisioning logic
type ContainerService struct {
	dockerClient        domain.DockerClient
	leaseRepository     domain.LeaseRepository
	containerRepository domain.ContainerRepository
	logger              *slog.Logger
	config              *config.Config
}

// ProvisionOptions captures a resource request
type ProvisionOptions struct {
	TenantID        string
	ImageType       string
	DurationMinutes int
	CPUMilli        int
	MemoryMB        int
	LogDemo         bool
	VolumeSizeMB    int
}

// NewContainerService creates a new container service
func NewContainerService(
	docker domain.DockerClient,
	leaseRepo domain.LeaseRepository,
	containerRepo domain.ContainerRepository,
	logger *slog.Logger,
	cfg *config.Config,
) *ContainerService {
	return &ContainerService{
		dockerClient:        docker,
		leaseRepository:     leaseRepo,
		containerRepository: containerRepo,
		logger:              logger,
		config:              cfg,
	}
}

// ProvisionContainer creates a new Docker container with a time-limited lease (async)
func (s *ContainerService) ProvisionContainer(ctx context.Context, opts ProvisionOptions) (*domain.Container, error) {
	// 1. Create domain entity with pending status
	now := time.Now()
	expiryTime := now.Add(time.Duration(opts.DurationMinutes) * time.Minute)

	container := &domain.Container{
		ID:          generateContainerID(), // Generate temp ID
		TenantID:    opts.TenantID,
		ImageType:   opts.ImageType,
		CPUMilli:    opts.CPUMilli,
		MemoryMB:    opts.MemoryMB,
		Status:      "pending", // Status is PENDING initially
		CreatedAt:   now,
		ExpiryAt:    expiryTime,
		MaxRestarts: 3, // Phase 2: Self-healing default max restarts
	}

	// 2. Store container in repository with pending status
	if err := s.containerRepository.Save(container); err != nil {
		return nil, fmt.Errorf("failed to save container: %w", err)
	}

	// 3. Create lease in Redis with TTL
	lease := &domain.Lease{
		ContainerID:     container.ID,
		LeaseKey:        fmt.Sprintf("lease:%s", container.ID),
		ExpiryTime:      expiryTime,
		DurationMinutes: opts.DurationMinutes,
		CreatedAt:       now,
	}
	if err := s.leaseRepository.CreateLease(lease); err != nil {
		_ = s.containerRepository.Delete(container.ID)
		return nil, fmt.Errorf("failed to create lease: %w", err)
	}

	// 4. Start async provisioning in background goroutine
	go s.asyncProvisionContainer(context.Background(), container.ID, opts.ImageType, opts.CPUMilli, opts.MemoryMB, opts.LogDemo, opts.VolumeSizeMB)

	return container, nil
}

// asyncProvisionContainer runs the actual Docker provisioning in background
func (s *ContainerService) asyncProvisionContainer(ctx context.Context, tempID string, imageType string, cpuMilli int, memoryMB int, logDemo bool, volumeSizeMB int) {
	s.logger.Info("starting async provisioning", slog.String("temp_id", tempID))
	start := time.Now()

	// Create volume if requested
	var volumeID string
	if volumeSizeMB > 0 {
		generatedVolumeID := fmt.Sprintf("vol-%s", tempID)
		volName, err := s.dockerClient.CreateVolume(ctx, generatedVolumeID, volumeSizeMB)
		if err != nil {
			s.logger.Error("failed to create volume",
				slog.String("temp_id", tempID),
				slog.String("volume_id", generatedVolumeID),
				slog.String("error", err.Error()),
			)
			metrics.ObserveProvision("error", time.Since(start))
			existingContainer, _ := s.containerRepository.GetByID(tempID)
			if existingContainer != nil {
				existingContainer.Status = "error"
				existingContainer.Error = fmt.Sprintf("failed to create volume: %v", err)
				_ = s.containerRepository.Save(existingContainer)
			}
			return
		}
		volumeID = volName
		s.logger.Info("volume created", slog.String("temp_id", tempID), slog.String("volume_id", volumeID))
	}

	// Create actual Docker container
	dockerID, err := s.dockerClient.CreateContainer(ctx, imageType, cpuMilli, memoryMB, logDemo, volumeID)
	if err != nil {
		s.logger.Error("failed to create container",
			slog.String("temp_id", tempID),
			slog.String("error", err.Error()),
		)
		metrics.ObserveProvision("error", time.Since(start))
		// Clean up volume if it was created
		if volumeID != "" {
			_ = s.dockerClient.RemoveVolume(context.Background(), volumeID)
		}
		// Mark as error and update
		existingContainer, _ := s.containerRepository.GetByID(tempID)
		if existingContainer != nil {
			existingContainer.Status = "error"
			existingContainer.Error = err.Error()
			_ = s.containerRepository.Save(existingContainer)
		}
		return
	}

	// Update container with real Docker ID, running status, and volume
	existingContainer, _ := s.containerRepository.GetByID(tempID)
	if existingContainer != nil {
		existingContainer.DockerID = dockerID
		existingContainer.Status = "running"
		existingContainer.VolumeID = volumeID
		existingContainer.VolumeSize = volumeSizeMB
		_ = s.containerRepository.Save(existingContainer)
		s.logger.Info("container successfully created", slog.String("temp_id", tempID), slog.String("docker_id", dockerID), slog.String("volume_id", volumeID))
		metrics.ObserveProvision("success", time.Since(start))
		metrics.IncrementActive()
	}
}

// GetContainer retrieves container details
func (s *ContainerService) GetContainer(ctx context.Context, containerID string) (*domain.Container, error) {
	container, err := s.containerRepository.GetByID(containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}
	return container, nil
}

// DeleteContainer manually removes a container (before expiry)
func (s *ContainerService) DeleteContainer(ctx context.Context, containerID string) error {
	// Get container to find Docker ID
	container, err := s.containerRepository.GetByID(containerID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}
	wasRunning := container.Status == "running"

	// Only try to stop/remove if we have a Docker ID (not still pending)
	if container.DockerID != "" {
		// Stop and remove from Docker
		if err := s.dockerClient.StopContainer(context.Background(), container.DockerID); err != nil {
			s.logger.Warn("failed to stop container", slog.String("docker_id", container.DockerID), slog.String("error", err.Error()))
		}
		if err := s.dockerClient.RemoveContainer(context.Background(), container.DockerID); err != nil {
			s.logger.Warn("failed to remove container", slog.String("docker_id", container.DockerID), slog.String("error", err.Error()))
		}
	}

	// Remove volume if attached
	if container.VolumeID != "" {
		if err := s.dockerClient.RemoveVolume(context.Background(), container.VolumeID); err != nil {
			s.logger.Warn("failed to remove volume", slog.String("container_id", containerID), slog.String("volume_id", container.VolumeID), slog.String("error", err.Error()))
		}
	}

	// Mark container as terminated
	container.Status = "terminated"
	container.ExpiryAt = time.Now().Add(15 * time.Minute) // retain record briefly

	if err := s.containerRepository.Save(container); err != nil {
		return fmt.Errorf("failed to persist container record: %w", err)
	}

	if err := s.leaseRepository.DeleteLease(fmt.Sprintf("lease:%s", containerID)); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	if wasRunning {
		metrics.DecrementActive()
	}
	metrics.ObserveCleanup("manual", "success")

	return nil
}

// generateContainerID generates a unique container ID
func generateContainerID() string {
	return fmt.Sprintf("container-%d", rand.Int63())
}
