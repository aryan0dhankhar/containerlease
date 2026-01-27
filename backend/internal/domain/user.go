package domain

import "time"

// User represents a system user
type User struct {
	ID           string // UUID
	Email        string // Unique email address
	Username     string // Unique username
	PasswordHash string // Bcrypt hashed password (not returned in API)
	TenantID     string // UUID of tenant this user belongs to
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsActive     bool
}

// UserRepository defines data access for users
type UserRepository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	ListByTenant(tenantID string) ([]*User, error)
}

// Tenant represents an organization/tenant
type Tenant struct {
	ID          string // UUID
	Name        string // Unique tenant name
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsActive    bool
}

// TenantRepository defines data access for tenants
type TenantRepository interface {
	Create(tenant *Tenant) error
	GetByID(id string) (*Tenant, error)
	GetByName(name string) (*Tenant, error)
	Update(tenant *Tenant) error
	Delete(id string) error
	List() ([]*Tenant, error)
}
