package api

import (
	"context"

	"saasbase/platform/authz"
	"saasbase/platform/module"
)

const (
	ResourceUser = "user.user"
	ResourceRole = "user.role"
)

func Permissions() []module.Permission {
	return []module.Permission{
		{Namespace: ResourceUser, Action: "read"},
		{Namespace: ResourceUser, Action: "write"},
		{Namespace: ResourceRole, Action: "read"},
		{Namespace: ResourceRole, Action: "write"},
	}
}

// PermissionManager allows managing tenant permissions and role assignments across modules.
type PermissionManager interface {
	SyncPermissions(ctx context.Context, tenantID string, perms []module.Permission) error
	AssignPermissions(ctx context.Context, roleID, tenantID string, perms []authz.Permission) error
	GetRolePermissions(ctx context.Context, roleID, tenantID string) ([]authz.Permission, error)
}
