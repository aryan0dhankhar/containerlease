package domain

import (
	"context"
	"io"
	"time"
)

// Container represents a Docker container entity
type Container struct {
	ID        string
	ImageType string
	Status    string // running, exited, stopped
	CreatedAt time.Time
	ExpiryAt  time.Time
}

// Lease represents a temporary lease/reservation for a container
type Lease struct {
	ContainerID     string
	LeaseKey        string // lease:{containerID}
	ExpiryTime      time.Time
	DurationMinutes int
	CreatedAt       time.Time
}

// ContainerRepository defines data access for containers
type ContainerRepository interface {
	GetByID(id string) (*Container, error)
	Save(container *Container) error
	Delete(id string) error
	List() ([]*Container, error)
}

// LeaseRepository defines data access for leases
type LeaseRepository interface {
	CreateLease(lease *Lease) error
	GetLease(leaseKey string) (*Lease, error)
	DeleteLease(leaseKey string) error
	GetExpiredLeases() ([]string, error) // Returns container IDs
}

// DockerClient defines Docker operations
type DockerClient interface {
	CreateContainer(ctx context.Context, imageType string) (string, error)
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
}
