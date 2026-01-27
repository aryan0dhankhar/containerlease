package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// User represents a system user
type User struct {
	ID       string
	Email    string
	Password string // hashed
	TenantID string
	Active   bool
}

// UserStore manages user authentication
type UserStore struct {
	mu    sync.RWMutex
	users map[string]*User // email -> user
}

// NewUserStore creates a new user store with demo users
func NewUserStore() *UserStore {
	store := &UserStore{
		users: make(map[string]*User),
	}

	// Add demo users for testing
	store.AddUser("demo@example.com", "demo123", "tenant-demo", "user-demo-1")
	store.AddUser("admin@example.com", "admin123", "tenant-admin", "user-admin-1")
	store.AddUser("test@example.com", "test123", "tenant-test", "user-test-1")

	return store
}

// AddUser adds a new user with hashed password
func (us *UserStore) AddUser(email, password, tenantID, userID string) {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.users[email] = &User{
		ID:       userID,
		Email:    email,
		Password: hashPassword(password),
		TenantID: tenantID,
		Active:   true,
	}
}

// Authenticate verifies credentials and returns user
func (us *UserStore) Authenticate(email, password string) (*User, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	user, exists := us.users[email]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	if !user.Active {
		return nil, fmt.Errorf("user inactive")
	}

	if user.Password != hashPassword(password) {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// GetUser retrieves a user by email
func (us *UserStore) GetUser(email string) (*User, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	user, exists := us.users[email]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// hashPassword creates a simple hash of the password
// In production, use bcrypt or argon2
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
