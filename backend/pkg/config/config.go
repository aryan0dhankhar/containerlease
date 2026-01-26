package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	Environment            string
	ServerPort             int
	RedisURL               string
	DockerHost             string
	CleanupIntervalMinutes int
	ContainerMaxDuration   int
	ContainerMinDuration   int
	LogLevel               string
	CORSAllowedOrigins     []string
	AllowedImages          []string
	DefaultCPUMilli        int
	MaxCPUMilli            int
	DefaultMemoryMB        int
	MaxMemoryMB            int
	MaxVolumeMB            int
	Presets                map[string]Preset
}

// Preset defines a provisioning template
type Preset struct {
	Name        string
	CPUMilli    int
	MemoryMB    int
	DurationMin int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}

	cleanupInterval, err := strconv.Atoi(getEnv("CLEANUP_INTERVAL_MINUTES", "1"))
	if err != nil {
		return nil, fmt.Errorf("invalid CLEANUP_INTERVAL_MINUTES: %w", err)
	}

	maxDuration, err := strconv.Atoi(getEnv("CONTAINER_MAX_DURATION_MINUTES", "120"))
	if err != nil {
		return nil, fmt.Errorf("invalid CONTAINER_MAX_DURATION_MINUTES: %w", err)
	}

	minDuration, err := strconv.Atoi(getEnv("CONTAINER_MIN_DURATION_MINUTES", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid CONTAINER_MIN_DURATION_MINUTES: %w", err)
	}

	defaultCPUMilli, err := strconv.Atoi(getEnv("DEFAULT_CPU_MILLI", "500"))
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_CPU_MILLI: %w", err)
	}

	maxCPUMilli, err := strconv.Atoi(getEnv("MAX_CPU_MILLI", "2000"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_CPU_MILLI: %w", err)
	}

	defaultMemoryMB, err := strconv.Atoi(getEnv("DEFAULT_MEMORY_MB", "512"))
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_MEMORY_MB: %w", err)
	}

	maxMemoryMB, err := strconv.Atoi(getEnv("MAX_MEMORY_MB", "2048"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_MEMORY_MB: %w", err)
	}

	maxVolumeMB, err := strconv.Atoi(getEnv("MAX_VOLUME_MB", "5120"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_VOLUME_MB: %w", err)
	}

	return &Config{
		Environment:            getEnv("ENVIRONMENT", "development"),
		ServerPort:             port,
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		DockerHost:             getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		CleanupIntervalMinutes: cleanupInterval,
		ContainerMaxDuration:   maxDuration,
		ContainerMinDuration:   minDuration,
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		CORSAllowedOrigins: []string{
			"http://localhost:5173",
			"http://localhost:3000",
		},
		AllowedImages:   parseCSVEnv("ALLOWED_IMAGES", []string{"ubuntu", "alpine"}),
		DefaultCPUMilli: defaultCPUMilli,
		MaxCPUMilli:     maxCPUMilli,
		DefaultMemoryMB: defaultMemoryMB,
		MaxMemoryMB:     maxMemoryMB,
		MaxVolumeMB:     maxVolumeMB,
		Presets: map[string]Preset{
			"tiny": {
				Name:        "Tiny (256MB, 250m CPU, 5min)",
				CPUMilli:    250,
				MemoryMB:    256,
				DurationMin: 5,
			},
			"standard": {
				Name:        "Standard (512MB, 500m CPU, 30min)",
				CPUMilli:    500,
				MemoryMB:    512,
				DurationMin: 30,
			},
			"large": {
				Name:        "Large (1GB, 1000m CPU, 60min)",
				CPUMilli:    1000,
				MemoryMB:    1024,
				DurationMin: 60,
			},
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseCSVEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return defaultValue
}
