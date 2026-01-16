package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client
type Client struct {
	cli    *client.Client
	logger *slog.Logger
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

	return &Client{cli: cli, logger: logger}, nil
}

// CreateContainer creates a new Docker container
func (c *Client) CreateContainer(ctx context.Context, imageType string) (string, error) {
	// Determine image name based on type
	var imageName string
	switch imageType {
	case "ubuntu":
		imageName = "ubuntu:latest"
	case "alpine":
		imageName = "alpine:latest"
	default:
		return "", fmt.Errorf("unsupported image type: %s", imageType)
	}

	// Pull image if needed
	if _, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container with resource limits
	config := &container.Config{
		Image: imageName,
		Cmd:   []string{"sleep", "infinity"}, // Keep container running
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:    512 * 1024 * 1024, // 512 MB
			CPUShares: 1024,              // Standard CPU share
		},
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
}

// StopContainer stops a running container
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10 // seconds
	options := container.StopOptions{Timeout: &timeout}
	return c.cli.ContainerStop(ctx, containerID, options)
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	options := container.RemoveOptions{Force: true}
	return c.cli.ContainerRemove(ctx, containerID, options)
}

// StreamLogs returns a channel that streams container logs
func (c *Client) StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	return c.cli.ContainerLogs(ctx, containerID, options)
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.cli.Close()
}
