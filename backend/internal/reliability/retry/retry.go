package retry

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Config holds retry strategy configuration
type Config struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// DefaultConfig returns sensible retry defaults
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// Retryable is a function that can be retried
type Retryable[T any] func(ctx context.Context) (T, error)

// Do executes a retryable function with exponential backoff
func Do[T any](ctx context.Context, cfg *Config, log *slog.Logger, op string, fn Retryable[T]) (T, error) {
	var zero T
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if attempt < cfg.MaxAttempts {
			backoff := calculateBackoff(attempt-1, cfg)
			log.Warn("operation failed, retrying",
				slog.String("operation", op),
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", cfg.MaxAttempts),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()),
			)
			time.Sleep(backoff)
		}
	}

	return zero, fmt.Errorf("operation '%s' failed after %d attempts: %w", op, cfg.MaxAttempts, lastErr)
}

// calculateBackoff returns exponential backoff duration
func calculateBackoff(attemptNum int, cfg *Config) time.Duration {
	backoff := time.Duration(float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attemptNum)))
	if backoff > cfg.MaxBackoff {
		backoff = cfg.MaxBackoff
	}
	return backoff
}
