package service

import (
	"errors"
	"testing"
	"time"

	"github.com/yourorg/containerlease/internal/domain"
)

type memUserRepo struct {
	byID       map[string]*domain.User
	byEmail    map[string]*domain.User
	byUsername map[string]*domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}, byUsername: map[string]*domain.User{}}
}

func (m *memUserRepo) Create(u *domain.User) error {
	if u.ID == "" {
		u.ID = "u-" + u.Email
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	m.byUsername[u.Username] = u
	return nil
}
func (m *memUserRepo) GetByID(id string) (*domain.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (m *memUserRepo) GetByEmail(email string) (*domain.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (m *memUserRepo) GetByUsername(username string) (*domain.User, error) {
	if u, ok := m.byUsername[username]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (m *memUserRepo) Update(u *domain.User) error {
	u.UpdatedAt = time.Now()
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	m.byUsername[u.Username] = u
	return nil
}
func (m *memUserRepo) Delete(id string) error { delete(m.byID, id); return nil }
func (m *memUserRepo) ListByTenant(tenantID string) ([]*domain.User, error) {
	out := []*domain.User{}
	for _, u := range m.byID {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out, nil
}

func TestRegisterAndLogin(t *testing.T) {
	repo := newMemUserRepo()
	s := NewAuthService(repo, "secret", nil)

	// Register
	r, err := s.Register("alice@example.com", "alice", "Password123", "tenant-1")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if r.UserID == "" || r.Token == "" {
		t.Fatalf("expected user id and token")
	}

	// Duplicate email
	if _, err := s.Register("alice@example.com", "alice2", "Password123", "tenant-1"); err == nil {
		t.Fatalf("expected duplicate email error")
	}

	// Login ok
	lr, err := s.Login("alice@example.com", "Password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if lr.Token == "" {
		t.Fatalf("expected token on login")
	}

	// Login wrong password
	if _, err := s.Login("alice@example.com", "Wrong"); err == nil {
		t.Fatalf("expected invalid credentials error")
	}
}

func TestChangePassword(t *testing.T) {
	repo := newMemUserRepo()
	s := NewAuthService(repo, "secret", nil)
	reg, err := s.Register("bob@example.com", "bob", "OldPass123", "tenant-1")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Wrong old password
	if err := s.ChangePassword(reg.UserID, "bad", "NewPass123"); err == nil {
		t.Fatalf("expected wrong old password error")
	}
	// Good change
	if err := s.ChangePassword(reg.UserID, "OldPass123", "NewPass123"); err != nil {
		t.Fatalf("change password failed: %v", err)
	}
	// Old password should no longer work
	if _, err := s.Login("bob@example.com", "OldPass123"); err == nil {
		t.Fatalf("expected old password to fail after change")
	}
	// New password works
	if _, err := s.Login("bob@example.com", "NewPass123"); err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}
