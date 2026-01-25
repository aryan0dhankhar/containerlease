package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/pkg/config"
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
	ImageType       string
	DurationMinutes int
	CPUMilli        int
	MemoryMB        int
	LogDemo         bool
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
	cost := calculateCost(opts.ImageType, float64(opts.DurationMinutes))

	container := &domain.Container{
		ID:        generateContainerID(), // Generate temp ID
		ImageType: opts.ImageType,
		CPUMilli:  opts.CPUMilli,
		MemoryMB:  opts.MemoryMB,
		Status:    "pending", // Status is PENDING initially
		CreatedAt: now,
		ExpiryAt:  expiryTime,
		Cost:      cost,
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
	go s.asyncProvisionContainer(context.Background(), container.ID, opts.ImageType, opts.CPUMilli, opts.MemoryMB, opts.LogDemo)

	return container, nil
}

// asyncProvisionContainer runs the actual Docker provisioning in background
func (s *ContainerService) asyncProvisionContainer(ctx context.Context, tempID string, imageType string, cpuMilli int, memoryMB int, logDemo bool) {
	s.logger.Info("starting async provisioning", slog.String("temp_id", tempID))

	// Create actual Docker container
	dockerID, err := s.dockerClient.CreateContainer(ctx, imageType, cpuMilli, memoryMB, logDemo)
	if err != nil {
		s.logger.Error("failed to create container",
			slog.String("temp_id", tempID),
			slog.String("error", err.Error()),
		)
		// Mark as error and update
		existingContainer, _ := s.containerRepository.GetByID(tempID)
		if existingContainer != nil {
			existingContainer.Status = "error"
			existingContainer.Error = err.Error()
			_ = s.containerRepository.Save(existingContainer)
		}
		return
	}

	// Update container with real Docker ID and running status
	existingContainer, _ := s.containerRepository.GetByID(tempID)
	if existingContainer != nil {
		existingContainer.DockerID = dockerID
		existingContainer.Status = "running"
		_ = s.containerRepository.Save(existingContainer)
		s.logger.Info("container successfully created", slog.String("temp_id", tempID), slog.String("docker_id", dockerID))
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

	// Compute usage-based cost at termination
	runtimeMinutes := time.Since(container.CreatedAt).Minutes()
	if runtimeMinutes < 0 {
		runtimeMinutes = 0
	}
	container.Cost = calculateCost(container.ImageType, runtimeMinutes)
	container.Status = "terminated"
	container.ExpiryAt = time.Now().Add(15 * time.Minute) // retain record briefly for billing visibility

	if err := s.containerRepository.Save(container); err != nil {
		return fmt.Errorf("failed to persist container record: %w", err)
	}

	if err := s.leaseRepository.DeleteLease(fmt.Sprintf("lease:%s", containerID)); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	return nil
}

// calculateCost returns the cost in dollars for a given instance type and duration
func calculateCost(imageType string, durationMinutes float64) float64 {
	hourlyRate := 0.0
	switch imageType {
	case "ubuntu":
		hourlyRate = 0.04 // $0.04/hour (Medium instance)
	case "alpine":
		hourlyRate = 0.01 // $0.01/hour (Small instance)
	default:
		hourlyRate = 0.04
	}
	durationHours := durationMinutes / 60.0
	return hourlyRate * durationHours
}

// generateContainerID generates a unique container ID
func generateContainerID() string {
	return fmt.Sprintf("container-%d", rand.Int63())
}
