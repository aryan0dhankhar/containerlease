package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
)

// ContainerService handles container provisioning logic
type ContainerService struct {
	dockerClient      domain.DockerClient
	leaseRepository   domain.LeaseRepository
	containerRepository domain.ContainerRepository
}

// NewContainerService creates a new container service
func NewContainerService(
	docker domain.DockerClient,
	leaseRepo domain.LeaseRepository,
	containerRepo domain.ContainerRepository,
) *ContainerService {
	return &ContainerService{
		dockerClient:       docker,
		leaseRepository:    leaseRepo,
		containerRepository: containerRepo,
	}
}

// ProvisionContainer creates a new Docker container with a time-limited lease
func (s *ContainerService) ProvisionContainer(ctx context.Context, imageType string, durationMinutes int) (*domain.Container, error) {
	// 1. Create Docker container
	containerID, err := s.dockerClient.CreateContainer(ctx, imageType)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// 2. Create domain entity
	now := time.Now()
	expiryTime := now.Add(time.Duration(durationMinutes) * time.Minute)
	container := &domain.Container{
		ID:        containerID,
		ImageType: imageType,
		Status:    "running",
		CreatedAt: now,
		ExpiryAt:  expiryTime,
	}

	// 3. Store container in repository
	if err := s.containerRepository.Save(container); err != nil {
		// Cleanup Docker container on repository error
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to save container: %w", err)
	}

	// 4. Create lease in Redis with TTL
	lease := &domain.Lease{
		ContainerID:     containerID,
		LeaseKey:        fmt.Sprintf("lease:%s", containerID),
		ExpiryTime:      expiryTime,
		DurationMinutes: durationMinutes,
		CreatedAt:       now,
	}
	if err := s.leaseRepository.CreateLease(lease); err != nil {
		// Cleanup on failure
		_ = s.containerRepository.Delete(containerID)
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to create lease: %w", err)
	}

	return container, nil
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
	// Stop and remove from Docker
	if err := s.dockerClient.StopContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	if err := s.dockerClient.RemoveContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Clean up from repositories
	if err := s.containerRepository.Delete(containerID); err != nil {
		return fmt.Errorf("failed to delete from repository: %w", err)
	}
	if err := s.leaseRepository.DeleteLease(fmt.Sprintf("lease:%s", containerID)); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	return nil
}
