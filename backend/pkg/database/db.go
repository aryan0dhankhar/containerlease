package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ConnectionPool manages database connections
type ConnectionPool struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewConnectionPool creates a new database connection pool
func NewConnectionPool(ctx context.Context, config *Config, logger *slog.Logger) (*ConnectionPool, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25) // default
	}

	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5) // default
	}

	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute) // default
	}

	// Test connection
	ctxTest, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTest); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connected successfully",
		slog.String("host", config.Host),
		slog.String("database", config.Database),
	)

	return &ConnectionPool{
		db:     db,
		logger: logger,
	}, nil
}

// GetDB returns the underlying sql.DB connection
func (cp *ConnectionPool) GetDB() *sql.DB {
	return cp.db
}

// Close closes the database connection
func (cp *ConnectionPool) Close() error {
	if cp.db != nil {
		return cp.db.Close()
	}
	return nil
}

// Health checks the database health
func (cp *ConnectionPool) Health(ctx context.Context) error {
	ctxTest, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return cp.db.PingContext(ctxTest)
}

// DefaultConfig returns default database configuration for development
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            5432,
		User:            "containerlease",
		Password:        "dev",
		Database:        "containerlease",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
}
