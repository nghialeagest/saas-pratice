package data

import (
	"context"

	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/data"
	"saasbase/platform/tenant"
)

type UserTenantRepo struct {
	provider *data.Provider
}

func NewUserTenantRepo(provider *data.Provider) *UserTenantRepo {
	return &UserTenantRepo{provider: provider}
}

func (r *UserTenantRepo) db(ctx context.Context) (*gorm.DB, error) {
	// UserTenant queries cross tenants, so tenant filter must be skipped
	return r.provider.DB(tenant.SkipFilter(ctx))
}

func (r *UserTenantRepo) Create(ctx context.Context, ut *biz.UserTenant) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Create(ut).Error
}

func (r *UserTenantRepo) Update(ctx context.Context, ut *biz.UserTenant) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Model(&biz.UserTenant{}).
		Where("user_id = ? AND tenant_id = ?", ut.UserID, ut.TenantID).
		Updates(map[string]any{
			"status":     ut.Status,
			"updated_at": ut.UpdatedAt,
		}).Error
}

func (r *UserTenantRepo) Get(ctx context.Context, userID, tenantID string) (*biz.UserTenant, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var ut biz.UserTenant
	if err := db.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&ut).Error; err != nil {
		return nil, err
	}
	return &ut, nil
}

func (r *UserTenantRepo) ListByUser(ctx context.Context, userID string) ([]*biz.UserTenant, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var list []*biz.UserTenant
	if err := db.Where("user_id = ?", userID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *UserTenantRepo) Delete(ctx context.Context, userID, tenantID string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND tenant_id = ?", userID, tenantID).Delete(&biz.UserTenant{}).Error
}
