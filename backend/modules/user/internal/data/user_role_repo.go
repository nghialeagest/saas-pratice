package data

import (
	"context"

	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/authz"
	"saasbase/platform/data"
	"saasbase/platform/tenant"
)

type UserRoleRepo struct {
	provider *data.Provider
	eventBus authz.EventBus
}

func NewUserRoleRepo(provider *data.Provider, bus authz.EventBus) *UserRoleRepo {
	if bus == nil {
		bus = authz.NewMemoryEventBus()
	}
	return &UserRoleRepo{
		provider: provider,
		eventBus: bus,
	}
}

func (r *UserRoleRepo) db(ctx context.Context) (*gorm.DB, error) {
	return r.provider.DB(ctx)
}

func (r *UserRoleRepo) Assign(ctx context.Context, userID string, roleIDs []string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	tid := tenant.FromContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {
		// Delete existing user roles in the current tenant
		if err := tx.Where("user_id = ? AND tenant_id = ?", userID, tid).Delete(&biz.UserRole{}).Error; err != nil {
			return err
		}
		// Insert new user roles
		for _, rid := range roleIDs {
			ur := &biz.UserRole{
				UserID:   userID,
				RoleID:   rid,
				TenantID: tid,
			}
			if err := tx.Create(ur).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Invalidate cached user permissions via event bus
	r.eventBus.Publish(ctx, authz.InvalidationEvent{
		Type:     authz.EventRoleAssigned,
		UserID:   userID,
		TenantID: tid,
	})

	return nil
}

func (r *UserRoleRepo) ListRoleIDsByUser(ctx context.Context, userID string) ([]string, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var roles []biz.UserRole
	if err := db.Where("user_id = ?", userID).Find(&roles).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(roles))
	for i, ur := range roles {
		ids[i] = ur.RoleID
	}
	return ids, nil
}

func (r *UserRoleRepo) ListRolesByUser(ctx context.Context, userID string) ([]*biz.Role, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*biz.Role
	err = db.Model(&biz.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id AND user_roles.tenant_id = roles.tenant_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
