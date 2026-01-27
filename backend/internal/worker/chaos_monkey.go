package worker

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
	"github.com/yourorg/containerlease/internal/observability/metrics"
)

// ChaosMonkey randomly kills containers to test system resilience (Phase 2)
// This helps verify that self-healing mechanisms work correctly
type ChaosMonkey struct {
	containerRepository domain.ContainerRepository
	dockerClient        domain.DockerClient
	logger              *slog.Logger
	interval            time.Duration
	killProbability     float64 // 0.0 to 1.0 - probability of killing a running container
	maxKillsPerCheck    int     // Maximum containers to kill in a single check
	enabled             bool    // Can be toggled on/off
}

// NewChaosMonkey creates a new chaos monkey worker
func NewChaosMonkey(
	containerRepo domain.ContainerRepository,
	dockerClient domain.DockerClient,
	logger *slog.Logger,
	interval time.Duration,
	killProbability float64,
	maxKillsPerCheck int,
) *ChaosMonkey {
	if killProbability < 0.0 {
		killProbability = 0.0
	}
	if killProbability > 1.0 {
		killProbability = 1.0
	}
	if maxKillsPerCheck < 1 {
		maxKillsPerCheck = 1
	}

	return &ChaosMonkey{
		containerRepository: containerRepo,
		dockerClient:        dockerClient,
		logger:              logger,
		interval:            interval,
		killProbability:     killProbability,
		maxKillsPerCheck:    maxKillsPerCheck,
		enabled:             false, // Disabled by default - must be explicitly enabled
	}
}

// SetEnabled toggles chaos monkey on/off
func (cm *ChaosMonkey) SetEnabled(enabled bool) {
	cm.enabled = enabled
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	cm.logger.Info("chaos monkey status changed", slog.String("status", status))
}

// Start begins the chaos monkey loop
func (cm *ChaosMonkey) Start(ctx context.Context) {
	ticker := time.NewTicker(cm.interval)
	defer ticker.Stop()

	cm.logger.Info("chaos monkey started",
		slog.Duration("interval", cm.interval),
		slog.Float64("kill_probability", cm.killProbability),
		slog.Int("max_kills_per_check", cm.maxKillsPerCheck),
	)

	for {
		select {
		case <-ctx.Done():
			cm.logger.Info("chaos monkey stopped")
			return
		case <-ticker.C:
			if cm.enabled {
				cm.injectChaos(ctx)
			}
		}
	}
}

// injectChaos performs a chaos check
func (cm *ChaosMonkey) injectChaos(ctx context.Context) {
	cm.logger.Debug("chaos monkey performing injection check")

	containers, err := cm.containerRepository.List()
	if err != nil {
		cm.logger.Error("failed to list containers for chaos injection",
			slog.String("error", err.Error()),
		)
		return
	}

	// Filter to only running containers
	var runningContainers []*domain.Container
	for _, c := range containers {
		if c.Status == "running" {
			runningContainers = append(runningContainers, c)
		}
	}

	if len(runningContainers) == 0 {
		cm.logger.Debug("no running containers to target for chaos")
		return
	}

	// Decide which containers to kill based on probability
	killCount := 0
	for _, container := range runningContainers {
		if killCount >= cm.maxKillsPerCheck {
			break
		}

		if rand.Float64() < cm.killProbability {
			cm.killContainer(ctx, container)
			killCount++
		}
	}

	if killCount > 0 {
		cm.logger.Info("chaos injection complete",
			slog.Int("containers_killed", killCount),
			slog.Int("total_running", len(runningContainers)),
		)
		metrics.ObserveChaosMoney("injection", killCount)
	}
}

// killContainer kills a single container
func (cm *ChaosMonkey) killContainer(ctx context.Context, container *domain.Container) {
	logger := cm.logger.With(
		slog.String("container_id", container.ID),
		slog.String("docker_id", container.DockerID),
		slog.String("image_type", container.ImageType),
	)

	logger.Info("chaos monkey killing container")

	// Kill the container by stopping it abruptly (no graceful shutdown)
	// This simulates a crash/failure
	if err := cm.dockerClient.RemoveContainer(ctx, container.DockerID); err != nil {
		logger.Error("failed to kill container",
			slog.String("error", err.Error()),
		)
		return
	}

	// Mark container as failed so cleanup worker can attempt restart
	container.Status = "exited"
	container.Error = "Killed by chaos monkey"
	container.LastFailureTime = time.Now()
	container.FailureReason = "Random termination for resilience testing"

	if err := cm.containerRepository.Save(container); err != nil {
		logger.Error("failed to update container status after kill",
			slog.String("error", err.Error()),
		)
		return
	}

	logger.Info("container killed by chaos monkey")
	metrics.ObserveChaosMoney("kill", 1)
}

// SetKillProbability updates the kill probability at runtime
func (cm *ChaosMonkey) SetKillProbability(probability float64) {
	if probability < 0.0 {
		probability = 0.0
	}
	if probability > 1.0 {
		probability = 1.0
	}
	cm.killProbability = probability
	cm.logger.Info("chaos monkey kill probability updated",
		slog.Float64("probability", cm.killProbability),
	)
}

// SetMaxKillsPerCheck updates the max kills per check interval at runtime
func (cm *ChaosMonkey) SetMaxKillsPerCheck(max int) {
	if max < 1 {
		max = 1
	}
	cm.maxKillsPerCheck = max
	cm.logger.Info("chaos monkey max kills per check updated",
		slog.Int("max_kills", cm.maxKillsPerCheck),
	)
}
