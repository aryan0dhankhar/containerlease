package security

import (
	"fmt"
	"log/slog"
)

// ResourceType identifies the kind of resource being accessed
type ResourceType string

const (
	ResourceContainer ResourceType = "container"
	ResourceSnapshot  ResourceType = "snapshot"
	ResourceUser      ResourceType = "user"
)

// Action identifies what operation is being performed
type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
)

// ResourcePermission checks fine-grained permissions on a specific resource
type ResourcePermission struct {
	ResourceType ResourceType
	ResourceID   string
	OwnerID      string // User ID that owns the resource
	Action       Action
}

// AuthorizationServiceV2 extends AuthorizationService with resource-level checks
type AuthorizationServiceV2 struct {
	logger *slog.Logger
}

// NewAuthorizationServiceV2 creates a new resource-aware authorization service
func NewAuthorizationServiceV2(logger *slog.Logger) *AuthorizationServiceV2 {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthorizationServiceV2{logger: logger}
}

// ValidateResourceAccess checks if a user has permission to access a specific resource.
// For now: only the owner or admins can access. Can be extended with delegation/sharing.
func (a *AuthorizationServiceV2) ValidateResourceAccess(
	userID string,
	role Role,
	perm ResourcePermission,
) error {
	// Admins bypass resource-level checks
	if role == RoleAdmin {
		return nil
	}

	// Only the owner can access resources they own
	if perm.OwnerID != userID {
		a.logger.Warn("resource access denied",
			slog.String("user_id", userID),
			slog.String("resource_id", perm.ResourceID),
			slog.String("resource_type", string(perm.ResourceType)),
			slog.String("owner_id", perm.OwnerID),
		)
		return fmt.Errorf("access denied: you do not own this %s", perm.ResourceType)
	}

	// Tenant admin and user can perform their allowed actions on owned resources
	return nil
}
