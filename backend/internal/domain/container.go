package domain

import (
	"context"
	"io"
	"time"
)

// Container represents a Docker container entity
type Container struct {
	ID              string // Our unique ID (not the Docker ID)
	DockerID        string // The actual Docker container ID
	TenantID        string // Tenant/User who owns this container
	ImageType       string
	Status          string // pending, running, exited, stopped, error
	CPUMilli        int    // Requested CPU in millicores
	MemoryMB        int    // Requested memory in MB
	CreatedAt       time.Time
	ExpiryAt        time.Time
	Cost            float64   // Cost in dollars
	Error           string    // Error message if status is error
	VolumeID        string    // Docker volume ID if volumes are attached
	VolumeSize      int       // Volume size in MB (0 if no volume)
	RestartCount    int       // Phase 2: Self-healing - number of restart attempts
	LastFailureTime time.Time // Phase 2: Self-healing - time of last failure
	FailureReason   string    // Phase 2: Self-healing - reason for last failure
	MaxRestarts     int       // Phase 2: Self-healing - maximum restart attempts (default: 3)
}

// Lease represents a temporary lease/reservation for a container
type Lease struct {
	ContainerID     string
	LeaseKey        string // lease:{containerID}
	ExpiryTime      time.Time
	DurationMinutes int
	CreatedAt       time.Time
}

// Snapshot represents a saved image of a container (Phase 2: Disaster Recovery)
type Snapshot struct {
	ID          string    // Unique snapshot ID
	ContainerID string    // Container this snapshot came from
	ImageName   string    // Docker image name (e.g., "localhost:5000/myapp:snapshot-uuid")
	CreatedAt   time.Time // When the snapshot was created
	Size        int64     // Snapshot size in bytes
	Description string    // Optional description/notes
	TenantID    string    // Tenant who owns this snapshot
}

// ContainerRepository defines data access for containers
type ContainerRepository interface {
	GetByID(id string) (*Container, error)
	Save(container *Container) error
	Delete(id string) error
	List() ([]*Container, error)
	ListByTenant(tenantID string) ([]*Container, error)
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
	StartContainer(ctx context.Context, containerID string) error
	StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
	CreateVolume(ctx context.Context, volumeID string, sizeMB int) (string, error)
	RemoveVolume(ctx context.Context, volumeID string) error
	// Phase 2: Snapshot/Disaster Recovery operations
	CommitContainer(ctx context.Context, containerID string, imageName string) error
	SaveImage(ctx context.Context, imageName string, filePath string) error
	LoadImage(ctx context.Context, filePath string) (string, error)
	RemoveImage(ctx context.Context, imageName string) error
}

// SnapshotRepository defines data access for snapshots
type SnapshotRepository interface {
	Create(snapshot *Snapshot) error
	GetByID(id string) (*Snapshot, error)
	GetByContainerID(containerID string) ([]*Snapshot, error)
	GetByTenant(tenantID string) ([]*Snapshot, error)
	Delete(id string) error
	DeleteByContainerID(containerID string) error
}
