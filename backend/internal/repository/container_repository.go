package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/internal/infrastructure/redis"
)

// ContainerRepository implements domain.ContainerRepository using Redis
type ContainerRepository struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewContainerRepository creates a new container repository
func NewContainerRepository(redisClient *redis.Client, logger *slog.Logger) *ContainerRepository {
	return &ContainerRepository{
		redis:  redisClient,
		logger: logger,
	}
}

// Save stores a container
func (r *ContainerRepository) Save(container *domain.Container) error {
	key := fmt.Sprintf("container:%s", container.ID)

	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container: %w", err)
	}

	// Apply TTL based on expiry to enforce lifecycle; ensure minimal TTL of 1 second
	ttl := time.Until(container.ExpiryAt)
	if ttl < time.Second {
		ttl = time.Second
	}

	if err := r.redis.Set(context.Background(), key, string(data), ttl); err != nil {
		return fmt.Errorf("failed to store container: %w", err)
	}

	r.logger.Debug("container saved", slog.String("container_id", container.ID))
	return nil
}

// GetByID retrieves a container by ID
func (r *ContainerRepository) GetByID(id string) (*domain.Container, error) {
	key := fmt.Sprintf("container:%s", id)

	data, err := r.redis.Get(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	var container domain.Container
	if err := json.Unmarshal([]byte(data), &container); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container: %w", err)
	}

	return &container, nil
}

// Delete removes a container
func (r *ContainerRepository) Delete(id string) error {
	key := fmt.Sprintf("container:%s", id)

	if err := r.redis.Delete(context.Background(), key); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	r.logger.Debug("container deleted", slog.String("container_id", id))
	return nil
}

// List returns all containers
func (r *ContainerRepository) List() ([]*domain.Container, error) {
	keys, err := r.redis.Keys(context.Background(), "container:*")
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []*domain.Container
	for _, key := range keys {
		data, err := r.redis.Get(context.Background(), key)
		if err != nil {
			r.logger.Error("failed to get container", slog.String("key", key), slog.String("error", err.Error()))
			continue
		}

		var c domain.Container
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			r.logger.Error("failed to unmarshal container", slog.String("key", key), slog.String("error", err.Error()))
			continue
		}

		// Ensure ID is present even if key parsing is needed
		if c.ID == "" {
			c.ID = strings.TrimPrefix(key, "container:")
		}

		containers = append(containers, &c)
	}

	return containers, nil
}
