package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/yourorg/containerlease/internal/domain"
)

// PostgresUserRepository implements domain.UserRepository using PostgreSQL
type PostgresUserRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresUserRepository creates a new user repository
func NewPostgresUserRepository(db *sql.DB, logger *slog.Logger) *PostgresUserRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresUserRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new user
func (r *PostgresUserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (email, username, password_hash, tenant_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.TenantID,
		user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create user",
			slog.String("email", user.Email),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(id string) (*domain.User, error) {
	user := &domain.User{}

	query := `
		SELECT id, email, username, password_hash, tenant_id, created_at, updated_at, is_active
		FROM users
		WHERE id = $1
	`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.TenantID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Error("failed to get user by id",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgresUserRepository) GetByEmail(email string) (*domain.User, error) {
	user := &domain.User{}

	query := `
		SELECT id, email, username, password_hash, tenant_id, created_at, updated_at, is_active
		FROM users
		WHERE email = $1 AND is_active = true
	`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.TenantID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (r *PostgresUserRepository) GetByUsername(username string) (*domain.User, error) {
	user := &domain.User{}

	query := `
		SELECT id, email, username, password_hash, tenant_id, created_at, updated_at, is_active
		FROM users
		WHERE username = $1 AND is_active = true
	`

	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.TenantID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// Update updates an existing user
func (r *PostgresUserRepository) Update(user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, username = $2, password_hash = $3, is_active = $4
		WHERE id = $5
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.IsActive,
		user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete soft-deletes a user (sets is_active to false)
func (r *PostgresUserRepository) Delete(id string) error {
	query := `
		UPDATE users
		SET is_active = false
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ListByTenant lists all users for a tenant
func (r *PostgresUserRepository) ListByTenant(tenantID string) ([]*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, tenant_id, created_at, updated_at, is_active
		FROM users
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		r.logger.Error("failed to list users by tenant",
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.TenantID,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.IsActive,
		)
		if err != nil {
			r.logger.Error("failed to scan user row",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}
