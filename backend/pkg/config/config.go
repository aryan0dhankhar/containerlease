package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Environment             string
	ServerPort              int
	RedisURL                string
	DockerHost              string
	CleanupIntervalMinutes  int
	ContainerMaxDuration    int
	ContainerMinDuration    int
	LogLevel                string
	CORSAllowedOrigins      []string
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
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
