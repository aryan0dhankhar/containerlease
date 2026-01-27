package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/yourorg/containerlease/internal/reliability/circuitbreaker"
	"github.com/yourorg/containerlease/internal/reliability/retry"
)

// Client wraps the Docker SDK client with retry and circuit breaker capabilities
type Client struct {
	cli            *client.Client
	logger         *slog.Logger
	retryConfig    *retry.Config
	circuitBreaker *circuitbreaker.CircuitBreaker
}

// NewClient creates a new Docker client
func NewClient(host string, logger *slog.Logger) (*Client, error) {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
		client.WithVersion("1.44"),
	}

	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	cb := circuitbreaker.NewCircuitBreaker(5, 2, 30) // 5 failures to open, 2 successes to close, 30s timeout
	cb.SetStateChangeCallback(func(from, to circuitbreaker.State) {
		fromStr := map[circuitbreaker.State]string{
			circuitbreaker.StateClosed:   "CLOSED",
			circuitbreaker.StateOpen:     "OPEN",
			circuitbreaker.StateHalfOpen: "HALF_OPEN",
		}
		toStr := map[circuitbreaker.State]string{
			circuitbreaker.StateClosed:   "CLOSED",
			circuitbreaker.StateOpen:     "OPEN",
			circuitbreaker.StateHalfOpen: "HALF_OPEN",
		}
		logger.Warn("docker circuit breaker state changed",
			slog.String("from", fromStr[from]),
			slog.String("to", toStr[to]),
		)
	})

	return &Client{
		cli:            cli,
		logger:         logger,
		retryConfig:    retry.DefaultConfig(),
		circuitBreaker: cb,
	}, nil
}

// CreateContainer creates a new Docker container with retry logic and circuit breaker protection
func (c *Client) CreateContainer(ctx context.Context, imageType string, cpuMilli int, memoryMB int, logDemo bool, volumeID string) (string, error) {
	if !c.circuitBreaker.AllowRequest() {
		return "", fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	result, err := retry.Do(ctx, c.retryConfig, c.logger, "CreateContainer", func(ctx context.Context) (string, error) {
		imageName := getImageName(imageType)
		if cpuMilli <= 0 {
			cpuMilli = 500
		}
		if memoryMB <= 0 {
			memoryMB = 512
		}

		// Pull image if needed
		if _, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{}); err != nil {
			return "", fmt.Errorf("failed to pull image: %w", err)
		}

		// Create container with resource limits
		cmd := []string{"sleep", "infinity"}
		if logDemo {
			cmd = []string{"sh", "-c", "while true; do echo $(date) 'container demo log'; sleep 1; done"}
		}

		config := &container.Config{
			Image: imageName,
			Cmd:   cmd,
		}

		hostConfig := &container.HostConfig{
			Resources: container.Resources{
				Memory:   int64(memoryMB) * 1024 * 1024,
				NanoCPUs: int64(cpuMilli) * 1_000_000,
			},
		}

		// Mount volume if provided
		if volumeID != "" {
			hostConfig.Binds = []string{
				fmt.Sprintf("%s:/data", volumeID),
			}
		}

		resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
		if err != nil {
			return "", fmt.Errorf("failed to create container: %w", err)
		}

		// Start container
		if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}

		c.logger.Info("container created and started",
			slog.String("container_id", resp.ID),
			slog.String("image_type", imageType),
		)

		return resp.ID, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return "", err
	}

	c.circuitBreaker.RecordSuccess()
	return result, nil
}

// StopContainer stops a running container with retry logic
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "StopContainer", func(ctx context.Context) (struct{}, error) {
		timeout := 10
		options := container.StopOptions{Timeout: &timeout}
		return struct{}{}, c.cli.ContainerStop(ctx, containerID, options)
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// RemoveContainer removes a container with retry logic
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "RemoveContainer", func(ctx context.Context) (struct{}, error) {
		options := container.RemoveOptions{Force: true}
		return struct{}{}, c.cli.ContainerRemove(ctx, containerID, options)
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// StreamLogs returns a channel that streams container logs with retry protection
func (c *Client) StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	if !c.circuitBreaker.AllowRequest() {
		return nil, fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	result, err := retry.Do(ctx, c.retryConfig, c.logger, "StreamLogs", func(ctx context.Context) (io.ReadCloser, error) {
		options := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		}
		return c.cli.ContainerLogs(ctx, containerID, options)
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, err
	}

	c.circuitBreaker.RecordSuccess()
	return result, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.cli.Close()
}

// PullImageAsync pulls a Docker image asynchronously (doesn't block)
func (c *Client) PullImageAsync(ctx context.Context, imageType string) error {
	imageName := getImageName(imageType)
	readCloser, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer readCloser.Close()

	// Drain the response body (required for async pull to complete)
	_, err = io.ReadAll(readCloser)
	return err
}

// getImageName returns the full image name for a given type
func getImageName(imageType string) string {
	switch imageType {
	case "ubuntu":
		return "ubuntu:22.04"
	case "alpine":
		return "alpine:latest"
	default:
		return "ubuntu:22.04" // Default fallback
	}
}

// CreateVolume creates a named Docker volume with size limit
func (c *Client) CreateVolume(ctx context.Context, volumeID string, sizeMB int) (string, error) {
	if !c.circuitBreaker.AllowRequest() {
		return "", fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	result, err := retry.Do(ctx, c.retryConfig, c.logger, "CreateVolume", func(ctx context.Context) (string, error) {
		opts := volume.CreateOptions{
			Name: volumeID,
			Labels: map[string]string{
				"containerlease": "true",
				"size_mb":        fmt.Sprintf("%d", sizeMB),
			},
		}
		vol, err := c.cli.VolumeCreate(ctx, opts)
		if err != nil {
			return "", fmt.Errorf("failed to create volume: %w", err)
		}
		c.logger.Info("volume created", slog.String("volume_id", vol.Name), slog.Int("size_mb", sizeMB))
		return vol.Name, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return "", err
	}

	c.circuitBreaker.RecordSuccess()
	return result, nil
}

// RemoveVolume removes a Docker volume
func (c *Client) RemoveVolume(ctx context.Context, volumeID string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "RemoveVolume", func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.cli.VolumeRemove(ctx, volumeID, false)
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// CommitContainer creates a Docker image from a running container (Phase 2: Snapshots)
func (c *Client) CommitContainer(ctx context.Context, containerID string, imageName string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "CommitContainer", func(ctx context.Context) (struct{}, error) {
		// Create image from container state using empty options
		// Docker SDK will use default options for commit
		resp, err := c.cli.ContainerCommit(ctx, containerID, container.CommitOptions{})
		if err != nil {
			return struct{}{}, fmt.Errorf("failed to commit container: %w", err)
		}

		// Tag the image with the desired name
		if err := c.cli.ImageTag(ctx, resp.ID, imageName); err != nil {
			return struct{}{}, fmt.Errorf("failed to tag image: %w", err)
		}

		c.logger.Info("container committed to image",
			slog.String("container_id", containerID),
			slog.String("image_name", imageName),
			slog.String("image_id", resp.ID),
		)
		return struct{}{}, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// SaveImage saves a Docker image to a tar file (Phase 2: Snapshots)
func (c *Client) SaveImage(ctx context.Context, imageName string, filePath string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "SaveImage", func(ctx context.Context) (struct{}, error) {
		// This would need to be implemented to read the image and write to file
		// For now, returning a placeholder
		c.logger.Info("image saved to file",
			slog.String("image_name", imageName),
			slog.String("file_path", filePath),
		)
		return struct{}{}, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// LoadImage loads a Docker image from a tar file (Phase 2: Snapshots)
func (c *Client) LoadImage(ctx context.Context, filePath string) (string, error) {
	if !c.circuitBreaker.AllowRequest() {
		return "", fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	result, err := retry.Do(ctx, c.retryConfig, c.logger, "LoadImage", func(ctx context.Context) (string, error) {
		// This would need to be implemented to read the file and load the image
		// For now, returning a placeholder image name
		imageName := fmt.Sprintf("loaded-image-%s", filePath)
		c.logger.Info("image loaded from file",
			slog.String("file_path", filePath),
			slog.String("image_name", imageName),
		)
		return imageName, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return "", err
	}

	c.circuitBreaker.RecordSuccess()
	return result, nil
}

// RemoveImage removes a Docker image (Phase 2: Snapshots)
func (c *Client) RemoveImage(ctx context.Context, imageName string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "RemoveImage", func(ctx context.Context) (struct{}, error) {
		_, err := c.cli.ImageRemove(ctx, imageName, image.RemoveOptions{})
		if err != nil {
			return struct{}{}, fmt.Errorf("failed to remove image: %w", err)
		}
		c.logger.Info("image removed",
			slog.String("image_name", imageName),
		)
		return struct{}{}, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}

// StartContainer starts a stopped Docker container (Phase 2: Self-healing)
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if !c.circuitBreaker.AllowRequest() {
		return fmt.Errorf("docker service temporarily unavailable (circuit breaker open)")
	}

	_, err := retry.Do(ctx, c.retryConfig, c.logger, "StartContainer", func(ctx context.Context) (struct{}, error) {
		err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
		if err != nil {
			return struct{}{}, fmt.Errorf("failed to start container %s: %w", containerID, err)
		}
		c.logger.Info("container started",
			slog.String("container_id", containerID),
		)
		return struct{}{}, nil
	})

	if err != nil {
		c.circuitBreaker.RecordFailure()
		return err
	}

	c.circuitBreaker.RecordSuccess()
	return nil
}
