package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aryan0dhankhar/containerlease/internal/domain"
	"github.com/aryan0dhankhar/containerlease/internal/handler"
	"github.com/aryan0dhankhar/containerlease/internal/security/middleware"
	"github.com/aryan0dhankhar/containerlease/internal/service"
)

// Mock implementations for testing

type mockContainerRepository struct {
	containers map[string]*domain.Container
}

func (m *mockContainerRepository) GetByID(id string) (*domain.Container, error) {
	if c, ok := m.containers[id]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("container not found")
}

func (m *mockContainerRepository) Save(container *domain.Container) error {
	m.containers[container.ID] = container
	return nil
}

func (m *mockContainerRepository) Delete(id string) error {
	delete(m.containers, id)
	return nil
}

func (m *mockContainerRepository) List() ([]*domain.Container, error) {
	var result []*domain.Container
	for _, c := range m.containers {
		result = append(result, c)
	}
	return result, nil
}

func (m *mockContainerRepository) ListByTenant(tenantID string) ([]*domain.Container, error) {
	var result []*domain.Container
	for _, c := range m.containers {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

type mockSnapshotRepository struct {
	snapshots map[string]*domain.Snapshot
}

func (m *mockSnapshotRepository) Create(snapshot *domain.Snapshot) error {
	m.snapshots[snapshot.ID] = snapshot
	return nil
}

func (m *mockSnapshotRepository) GetByID(id string) (*domain.Snapshot, error) {
	if s, ok := m.snapshots[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("snapshot not found")
}

func (m *mockSnapshotRepository) GetByContainerID(containerID string) ([]*domain.Snapshot, error) {
	var result []*domain.Snapshot
	for _, s := range m.snapshots {
		if s.ContainerID == containerID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSnapshotRepository) GetByTenant(tenantID string) ([]*domain.Snapshot, error) {
	var result []*domain.Snapshot
	for _, s := range m.snapshots {
		if s.TenantID == tenantID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSnapshotRepository) Delete(id string) error {
	delete(m.snapshots, id)
	return nil
}

func (m *mockSnapshotRepository) DeleteByContainerID(containerID string) error {
	for id, s := range m.snapshots {
		if s.ContainerID == containerID {
			delete(m.snapshots, id)
		}
	}
	return nil
}

type mockDockerClient struct {
	committedContainers map[string]string
	removedImages       map[string]bool
}

func (m *mockDockerClient) CreateContainer(ctx context.Context, imageType string, cpuMilli int, memoryMB int, logDemo bool, volumeID string) (string, error) {
	return "docker-id-123", nil
}

func (m *mockDockerClient) StopContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockDockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockDockerClient) StartContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockDockerClient) StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("mock logs"))), nil
}

func (m *mockDockerClient) CreateVolume(ctx context.Context, volumeID string, sizeMB int) (string, error) {
	return volumeID, nil
}

func (m *mockDockerClient) RemoveVolume(ctx context.Context, volumeID string) error {
	return nil
}

func (m *mockDockerClient) CommitContainer(ctx context.Context, containerID string, imageName string) error {
	m.committedContainers[containerID] = imageName
	return nil
}

func (m *mockDockerClient) SaveImage(ctx context.Context, imageName string, filePath string) error {
	return nil
}

func (m *mockDockerClient) LoadImage(ctx context.Context, filePath string) (string, error) {
	return "loaded-image-id", nil
}

func (m *mockDockerClient) RemoveImage(ctx context.Context, imageName string) error {
	m.removedImages[imageName] = true
	return nil
}

// TestCreateSnapshot tests creating a snapshot of a running container
func TestCreateSnapshot(t *testing.T) {
	logger := slog.Default()
	containerRepo := &mockContainerRepository{
		containers: make(map[string]*domain.Container),
	}
	snapshotRepo := &mockSnapshotRepository{
		snapshots: make(map[string]*domain.Snapshot),
	}
	dockerClient := &mockDockerClient{
		committedContainers: make(map[string]string),
		removedImages:       make(map[string]bool),
	}

	// Create a running container
	container := &domain.Container{
		ID:        "container-123",
		DockerID:  "docker-123",
		TenantID:  "tenant-456",
		ImageType: "ubuntu",
		Status:    "running",
		CreatedAt: time.Now(),
		ExpiryAt:  time.Now().Add(30 * time.Minute),
	}
	containerRepo.Save(container)

	snapshotService := service.NewSnapshotService(
		dockerClient,
		containerRepo,
		snapshotRepo,
		logger,
		nil,
	)

	snapshotHandler := handler.NewSnapshotHandler(
		snapshotService,
		containerRepo,
		logger,
	)

	// Create a mux with the snapshot routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/containers/{id}/snapshot", snapshotHandler.CreateSnapshot)

	// Create request
	reqBody := handler.CreateSnapshotRequest{
		SnapshotName: "backup-v1",
		Description:  "Production backup",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/containers/container-123/snapshot",
		bytes.NewReader(bodyBytes),
	)

	// Add tenant context
	ctx := middleware.SetTenantInContext(req.Context(), "tenant-456")
	req = req.WithContext(ctx)

	// Create response writer
	w := httptest.NewRecorder()

	// Serve through mux
	mux.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp handler.SnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ContainerID != "container-123" {
		t.Errorf("expected container ID container-123, got %s", resp.ContainerID)
	}

	if resp.Description != "Production backup" {
		t.Errorf("expected description 'Production backup', got %s", resp.Description)
	}

	// Verify Docker commit was called
	if imageName, ok := dockerClient.committedContainers["docker-123"]; !ok {
		t.Error("expected docker.CommitContainer to be called")
	} else {
		if imageName == "" {
			t.Error("expected non-empty image name")
		}
	}
}

// TestCreateSnapshotUnauthorized tests that only container owner can snapshot
func TestCreateSnapshotUnauthorized(t *testing.T) {
	logger := slog.Default()
	containerRepo := &mockContainerRepository{
		containers: make(map[string]*domain.Container),
	}
	snapshotRepo := &mockSnapshotRepository{
		snapshots: make(map[string]*domain.Snapshot),
	}
	dockerClient := &mockDockerClient{
		committedContainers: make(map[string]string),
		removedImages:       make(map[string]bool),
	}

	// Create container owned by different tenant
	container := &domain.Container{
		ID:        "container-123",
		DockerID:  "docker-123",
		TenantID:  "tenant-original",
		ImageType: "ubuntu",
		Status:    "running",
		CreatedAt: time.Now(),
		ExpiryAt:  time.Now().Add(30 * time.Minute),
	}
	containerRepo.Save(container)

	snapshotService := service.NewSnapshotService(
		dockerClient,
		containerRepo,
		snapshotRepo,
		logger,
		nil,
	)

	snapshotHandler := handler.NewSnapshotHandler(
		snapshotService,
		containerRepo,
		logger,
	)

	// Create a mux with the snapshot routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/containers/{id}/snapshot", snapshotHandler.CreateSnapshot)

	// Create request as different tenant
	reqBody := handler.CreateSnapshotRequest{
		SnapshotName: "backup-v1",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/containers/container-123/snapshot",
		bytes.NewReader(bodyBytes),
	)

	// Add different tenant context
	ctx := middleware.SetTenantInContext(req.Context(), "tenant-attacker")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should be forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Verify Docker commit was NOT called
	if _, ok := dockerClient.committedContainers["docker-123"]; ok {
		t.Error("expected docker.CommitContainer NOT to be called for unauthorized tenant")
	}
}

// TestListSnapshots tests listing snapshots for a container
func TestListSnapshots(t *testing.T) {
	logger := slog.Default()
	containerRepo := &mockContainerRepository{
		containers: make(map[string]*domain.Container),
	}
	snapshotRepo := &mockSnapshotRepository{
		snapshots: make(map[string]*domain.Snapshot),
	}
	dockerClient := &mockDockerClient{
		committedContainers: make(map[string]string),
		removedImages:       make(map[string]bool),
	}

	// Create container
	container := &domain.Container{
		ID:       "container-123",
		TenantID: "tenant-456",
		Status:   "running",
	}
	containerRepo.Save(container)

	// Create some snapshots
	snap1 := &domain.Snapshot{
		ID:          "snapshot-1",
		ContainerID: "container-123",
		TenantID:    "tenant-456",
		ImageName:   "backup-1",
		CreatedAt:   time.Now(),
	}
	snap2 := &domain.Snapshot{
		ID:          "snapshot-2",
		ContainerID: "container-123",
		TenantID:    "tenant-456",
		ImageName:   "backup-2",
		CreatedAt:   time.Now(),
	}
	snapshotRepo.Create(snap1)
	snapshotRepo.Create(snap2)

	snapshotService := service.NewSnapshotService(
		dockerClient,
		containerRepo,
		snapshotRepo,
		logger,
		nil,
	)

	snapshotHandler := handler.NewSnapshotHandler(
		snapshotService,
		containerRepo,
		logger,
	)

	// Create a mux with the snapshot routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/containers/{id}/snapshots", snapshotHandler.ListSnapshots)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/containers/container-123/snapshots",
		nil,
	)

	// Add tenant context
	ctx := middleware.SetTenantInContext(req.Context(), "tenant-456")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var respList []handler.SnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &respList); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(respList) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(respList))
	}
}

// TestDeleteSnapshot tests deleting a snapshot
func TestDeleteSnapshot(t *testing.T) {
	logger := slog.Default()
	containerRepo := &mockContainerRepository{
		containers: make(map[string]*domain.Container),
	}
	snapshotRepo := &mockSnapshotRepository{
		snapshots: make(map[string]*domain.Snapshot),
	}
	dockerClient := &mockDockerClient{
		committedContainers: make(map[string]string),
		removedImages:       make(map[string]bool),
	}

	// Create snapshot
	snap := &domain.Snapshot{
		ID:          "snapshot-123",
		ContainerID: "container-456",
		TenantID:    "tenant-789",
		ImageName:   "backup-image",
		CreatedAt:   time.Now(),
	}
	snapshotRepo.Create(snap)

	snapshotService := service.NewSnapshotService(
		dockerClient,
		containerRepo,
		snapshotRepo,
		logger,
		nil,
	)

	snapshotHandler := handler.NewSnapshotHandler(
		snapshotService,
		containerRepo,
		logger,
	)

	// Create a mux with the snapshot routes
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/snapshots/{id}", snapshotHandler.DeleteSnapshot)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/snapshots/snapshot-123",
		nil,
	)

	// Add tenant context
	ctx := middleware.SetTenantInContext(req.Context(), "tenant-789")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify snapshot was deleted
	if _, err := snapshotRepo.GetByID("snapshot-123"); err == nil {
		t.Error("expected snapshot to be deleted")
	}

	// Verify Docker image was removed
	if !dockerClient.removedImages["backup-image"] {
		t.Error("expected Docker image to be removed")
	}
}
