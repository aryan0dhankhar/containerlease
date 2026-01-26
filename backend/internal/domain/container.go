package domain

import (
	"context"
	"io"
	"time"
)

// Container represents a Docker container entity
type Container struct {
	ID         string // Our unique ID (not the Docker ID)
	DockerID   string // The actual Docker container ID
	ImageType  string
	Status     string // pending, running, exited, stopped, error
	CPUMilli   int    // Requested CPU in millicores
	MemoryMB   int    // Requested memory in MB
	CreatedAt  time.Time
	ExpiryAt   time.Time
	Cost       float64 // Cost in dollars
	Error      string  // Error message if status is error
	VolumeID   string  // Docker volume ID if volumes are attached
	VolumeSize int     // Volume size in MB (0 if no volume)
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
	CreateContainer(ctx context.Context, imageType string, cpuMilli int, memoryMB int, logDemo bool, volumeID string) (string, error)
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
	CreateVolume(ctx context.Context, volumeID string, sizeMB int) (string, error)
	RemoveVolume(ctx context.Context, volumeID string) error
}
