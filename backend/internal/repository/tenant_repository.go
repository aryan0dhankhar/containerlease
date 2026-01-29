package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aryan0dhankhar/containerlease/internal/domain"
)

// PostgresTenantRepository implements domain.TenantRepository using PostgreSQL
type PostgresTenantRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresTenantRepository creates a new tenant repository
func NewPostgresTenantRepository(db *sql.DB, logger *slog.Logger) *PostgresTenantRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresTenantRepository{db: db, logger: logger}
}

// Create creates a new tenant
func (r *PostgresTenantRepository) Create(tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query, tenant.Name, tenant.Description, tenant.IsActive).Scan(
		&tenant.ID,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)
}

// GetByID retrieves a tenant by ID
func (r *PostgresTenantRepository) GetByID(id string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	query := `
		SELECT id, name, description, created_at, updated_at, is_active
		FROM tenants
		WHERE id = $1
	`
	err := r.db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.CreatedAt, &t.UpdatedAt, &t.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return t, nil
}

// GetByName retrieves a tenant by name
func (r *PostgresTenantRepository) GetByName(name string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	query := `
		SELECT id, name, description, created_at, updated_at, is_active
		FROM tenants
		WHERE name = $1
	`
	err := r.db.QueryRow(query, name).Scan(
		&t.ID, &t.Name, &t.Description, &t.CreatedAt, &t.UpdatedAt, &t.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant by name: %w", err)
	}
	return t, nil
}

// Update updates an existing tenant
func (r *PostgresTenantRepository) Update(tenant *domain.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $1, description = $2, is_active = $3
		WHERE id = $4
		RETURNING updated_at
	`
	return r.db.QueryRow(query, tenant.Name, tenant.Description, tenant.IsActive, tenant.ID).Scan(&tenant.UpdatedAt)
}

// Delete soft-deletes a tenant (sets is_active=false)
func (r *PostgresTenantRepository) Delete(id string) error {
	query := `
		UPDATE tenants SET is_active=false WHERE id=$1
	`
	res, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

// List returns all tenants
func (r *PostgresTenantRepository) List() ([]*domain.Tenant, error) {
	query := `
		SELECT id, name, description, created_at, updated_at, is_active
		FROM tenants
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var out []*domain.Tenant
	for rows.Next() {
		t := &domain.Tenant{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedAt, &t.UpdatedAt, &t.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
