package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourorg/containerlease/internal/domain"
)

// SnapshotRepository implements domain.SnapshotRepository using Redis
type SnapshotRepository struct {
	client *redis.Client
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(client *redis.Client) *SnapshotRepository {
	return &SnapshotRepository{
		client: client,
	}
}

// Create saves a new snapshot
func (r *SnapshotRepository) Create(snapshot *domain.Snapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("snapshot:%s", snapshot.ID)
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Store snapshot metadata with TTL of 30 days
	ttl := 30 * 24 * time.Hour
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store snapshot: %w", err)
	}

	// Add to container's snapshot list
	containerKey := fmt.Sprintf("container_snapshots:%s", snapshot.ContainerID)
	if err := r.client.SAdd(ctx, containerKey, snapshot.ID).Err(); err != nil {
		return fmt.Errorf("failed to add snapshot to container list: %w", err)
	}

	// Add to tenant's snapshot list
	tenantKey := fmt.Sprintf("tenant_snapshots:%s", snapshot.TenantID)
	if err := r.client.SAdd(ctx, tenantKey, snapshot.ID).Err(); err != nil {
		return fmt.Errorf("failed to add snapshot to tenant list: %w", err)
	}

	return nil
}

// GetByID retrieves a snapshot by ID
func (r *SnapshotRepository) GetByID(id string) (*domain.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("snapshot:%s", id)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	var snapshot domain.Snapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetByContainerID retrieves all snapshots for a container
func (r *SnapshotRepository) GetByContainerID(containerID string) ([]*domain.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerKey := fmt.Sprintf("container_snapshots:%s", containerID)
	snapshotIDs, err := r.client.SMembers(ctx, containerKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot IDs: %w", err)
	}

	var snapshots []*domain.Snapshot
	for _, id := range snapshotIDs {
		snapshot, err := r.GetByID(id)
		if err != nil {
			// Log error but continue with other snapshots
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetByTenant retrieves all snapshots for a tenant
func (r *SnapshotRepository) GetByTenant(tenantID string) ([]*domain.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tenantKey := fmt.Sprintf("tenant_snapshots:%s", tenantID)
	snapshotIDs, err := r.client.SMembers(ctx, tenantKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot IDs: %w", err)
	}

	var snapshots []*domain.Snapshot
	for _, id := range snapshotIDs {
		snapshot, err := r.GetByID(id)
		if err != nil {
			// Log error but continue with other snapshots
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// Delete removes a snapshot
func (r *SnapshotRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("snapshot:%s", id)

	// Get snapshot first to remove from container/tenant lists
	snapshot, err := r.GetByID(id)
	if err != nil {
		return err
	}

	// Remove from container's list
	containerKey := fmt.Sprintf("container_snapshots:%s", snapshot.ContainerID)
	if err := r.client.SRem(ctx, containerKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from container list: %w", err)
	}

	// Remove from tenant's list
	tenantKey := fmt.Sprintf("tenant_snapshots:%s", snapshot.TenantID)
	if err := r.client.SRem(ctx, tenantKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from tenant list: %w", err)
	}

	// Remove snapshot data
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	return nil
}

// DeleteByContainerID removes all snapshots for a container
func (r *SnapshotRepository) DeleteByContainerID(containerID string) error {
	snapshots, err := r.GetByContainerID(containerID)
	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		if err := r.Delete(snapshot.ID); err != nil {
			return err
		}
	}

	return nil
}
