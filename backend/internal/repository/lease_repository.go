package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/internal/infrastructure/redis"
)

// LeaseRepository implements domain.LeaseRepository using Redis
type LeaseRepository struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewLeaseRepository creates a new lease repository
func NewLeaseRepository(redisClient *redis.Client, logger *slog.Logger) *LeaseRepository {
	return &LeaseRepository{
		redis:  redisClient,
		logger: logger,
	}
}

// CreateLease stores a lease in Redis with TTL
func (r *LeaseRepository) CreateLease(lease *domain.Lease) error {
	data, err := json.Marshal(lease)
	if err != nil {
		return fmt.Errorf("failed to marshal lease: %w", err)
	}

	// Calculate TTL based on expiry time
	ttl := time.Until(lease.ExpiryTime)
	if ttl <= 0 {
		ttl = time.Second // Minimum TTL
	}

	if err := r.redis.Set(context.Background(), lease.LeaseKey, string(data), ttl); err != nil {
		return fmt.Errorf("failed to store lease: %w", err)
	}

	r.logger.Debug("lease created", slog.String("lease_key", lease.LeaseKey))
	return nil
}

// GetLease retrieves a lease from Redis
func (r *LeaseRepository) GetLease(leaseKey string) (*domain.Lease, error) {
	data, err := r.redis.Get(context.Background(), leaseKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get lease: %w", err)
	}

	var lease domain.Lease
	if err := json.Unmarshal([]byte(data), &lease); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lease: %w", err)
	}

	return &lease, nil
}

// DeleteLease removes a lease from Redis
func (r *LeaseRepository) DeleteLease(leaseKey string) error {
	if err := r.redis.Delete(context.Background(), leaseKey); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}
	return nil
}

// GetExpiredLeases returns all container IDs with expired leases
func (r *LeaseRepository) GetExpiredLeases() ([]string, error) {
	// Get all lease keys
	keys, err := r.redis.Keys(context.Background(), "lease:*")
	if err != nil {
		return nil, fmt.Errorf("failed to get lease keys: %w", err)
	}

	var expiredContainerIDs []string

	for _, key := range keys {
		ttl, err := r.redis.TTL(context.Background(), key)
		if err != nil {
			r.logger.Error("failed to get ttl", slog.String("key", key), slog.String("error", err.Error()))
			continue
		}

		// TTL < 0 means key doesn't exist or has no expiry
		// But we set TTL when creating, so if TTL is 0 or negative (redis returns negative for expired)
		// it means it's expired or about to expire
		if ttl <= 0 {
			// Extract container ID from "lease:abc123"
			containerID := key[6:] // Remove "lease:" prefix
			expiredContainerIDs = append(expiredContainerIDs, containerID)
		}
	}

	return expiredContainerIDs, nil
}
