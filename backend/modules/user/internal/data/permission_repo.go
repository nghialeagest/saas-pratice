package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/authz"
	"saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

type PermissionRepo struct {
	provider *data.Provider
	eventBus authz.EventBus
}

func NewPermissionRepo(provider *data.Provider, bus authz.EventBus) *PermissionRepo {
	if bus == nil {
		bus = authz.NewMemoryEventBus()
	}
	return &PermissionRepo{
		provider: provider,
		eventBus: bus,
	}
}

func (r *PermissionRepo) db(ctx context.Context) (*gorm.DB, error) {
	return r.provider.DB(ctx)
}

// SyncPermissions synchronizes system permissions for a tenant.
func (r *PermissionRepo) SyncPermissions(ctx context.Context, tenantID string, perms []module.Permission) error {
	db, err := r.db(tenant.SkipFilter(ctx))
	if err != nil {
		return err
	}

	records := make([]biz.Permission, 0, len(perms))
	for _, p := range perms {
		records = append(records, biz.Permission{
			ID:        uuid.NewString(),
			TenantID:  tenantID,
			Namespace: p.Namespace,
			Action:    p.Action,
			CreatedAt: time.Now(),
		})
	}

	if len(records) == 0 {
		return nil
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "namespace"}, {Name: "action"}},
		DoNothing: true,
	}).Create(&records).Error
}

// AssignPermissions assigns a set of permissions to a role within a tenant and invalidates cache.
func (r *PermissionRepo) AssignPermissions(ctx context.Context, roleID, tenantID string, perms []authz.Permission) error {
	db, err := r.db(tenant.SkipFilter(ctx))
	if err != nil {
		return err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ? AND tenant_id = ?", roleID, tenantID).Delete(&biz.RolePermission{}).Error; err != nil {
			return err
		}

		if len(perms) == 0 {
			return nil
		}

		records := make([]biz.RolePermission, 0, len(perms))
		for _, p := range perms {
			records = append(records, biz.RolePermission{
				RoleID:    roleID,
				TenantID:  tenantID,
				Namespace: p.Namespace,
				Action:    p.Action,
				CreatedAt: time.Now(),
			})
		}
		return tx.Create(&records).Error
	})
	if err != nil {
		return err
	}

	// Dispatch invalidation event to invalidate cached entries for this tenant's role
	r.eventBus.Publish(ctx, authz.InvalidationEvent{
		Type:     authz.EventRolePermsChanged,
		RoleID:   roleID,
		TenantID: tenantID,
	})

	return nil
}

// GetUserPermissions queries all permissions assigned to a user's roles within a tenant.
// Implements authz.PermissionSource.
func (r *PermissionRepo) GetUserPermissions(ctx context.Context, userID, tenantID string) ([]authz.Permission, error) {
	db, err := r.db(tenant.SkipFilter(ctx))
	if err != nil {
		return nil, err
	}

	type result struct {
		Namespace string
		Action    string
	}
	var results []result

	err = db.Table("role_permissions").
		Select("role_permissions.namespace, role_permissions.action").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id AND user_roles.tenant_id = role_permissions.tenant_id").
		Where("user_roles.user_id = ? AND user_roles.tenant_id = ?", userID, tenantID).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	perms := make([]authz.Permission, len(results))
	for i, r := range results {
		perms[i] = authz.Permission{
			Namespace: r.Namespace,
			Action:    r.Action,
		}
	}
	return perms, nil
}

// GetRolePermissions returns the permissions granted to a single role.
func (r *PermissionRepo) GetRolePermissions(ctx context.Context, roleID, tenantID string) ([]authz.Permission, error) {
	db, err := r.db(tenant.SkipFilter(ctx))
	if err != nil {
		return nil, err
	}

	type result struct {
		Namespace string
		Action    string
	}
	var results []result

	err = db.Table("role_permissions").
		Select("namespace, action").
		Where("role_id = ? AND tenant_id = ?", roleID, tenantID).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	perms := make([]authz.Permission, len(results))
	for i, row := range results {
		perms[i] = authz.Permission{Namespace: row.Namespace, Action: row.Action}
	}
	return perms, nil
}
