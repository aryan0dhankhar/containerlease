package security

import (
	"fmt"
	"log/slog"
)

// Role represents a user role
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleTenantAdmin Role = "tenant_admin"
	RoleUser        Role = "user"
)

// Permission represents an action permission
type Permission string

const (
	PermCreateContainer Permission = "create_container"
	PermDeleteContainer Permission = "delete_container"
	PermReadContainer   Permission = "read_container"
	PermListContainers  Permission = "list_containers"
	PermCreateSnapshot  Permission = "create_snapshot"
	PermDeleteSnapshot  Permission = "delete_snapshot"
	PermListSnapshots   Permission = "list_snapshots"
	PermManageUsers     Permission = "manage_users"
	PermManageTenant    Permission = "manage_tenant"
	PermViewAuditLog    Permission = "view_audit_log"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermCreateContainer,
		PermDeleteContainer,
		PermReadContainer,
		PermListContainers,
		PermCreateSnapshot,
		PermDeleteSnapshot,
		PermListSnapshots,
		PermManageUsers,
		PermManageTenant,
		PermViewAuditLog,
	},
	RoleTenantAdmin: {
		PermCreateContainer,
		PermDeleteContainer,
		PermReadContainer,
		PermListContainers,
		PermCreateSnapshot,
		PermDeleteSnapshot,
		PermListSnapshots,
		PermManageUsers,
		PermViewAuditLog,
	},
	RoleUser: {
		PermCreateContainer,
		PermDeleteContainer,
		PermReadContainer,
		PermListContainers,
		PermCreateSnapshot,
		PermDeleteSnapshot,
		PermListSnapshots,
	},
}

// AuthorizationService handles authorization checks
type AuthorizationService struct {
	logger *slog.Logger
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(logger *slog.Logger) *AuthorizationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthorizationService{
		logger: logger,
	}
}

// HasPermission checks if a role has a specific permission
func (as *AuthorizationService) HasPermission(role Role, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// ValidatePermission validates that a role has a specific permission
func (as *AuthorizationService) ValidatePermission(role Role, permission Permission) error {
	if !as.HasPermission(role, permission) {
		as.logger.Warn("permission denied",
			slog.String("role", string(role)),
			slog.String("permission", string(permission)),
		)
		return fmt.Errorf("permission denied: %s role cannot %s", role, permission)
	}
	return nil
}

// GetRolePermissions returns all permissions for a role
func (as *AuthorizationService) GetRolePermissions(role Role) []Permission {
	return RolePermissions[role]
}

// ValidateTenantAccess checks if a user belongs to a tenant
func (as *AuthorizationService) ValidateTenantAccess(userTenantID, requestedTenantID string) error {
	if userTenantID != requestedTenantID {
		as.logger.Warn("tenant access denied",
			slog.String("user_tenant", userTenantID),
			slog.String("requested_tenant", requestedTenantID),
		)
		return fmt.Errorf("access denied: invalid tenant")
	}
	return nil
}
