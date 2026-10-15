package data

import (
	"context"

	"gorm.io/gorm"

	"saasbase/modules/tenant/internal/biz"
	"saasbase/platform/data"
	"saasbase/platform/tenant"
)

type TenantRepo struct {
	provider *data.Provider
}

var _ biz.Repo = (*TenantRepo)(nil)

func NewTenantRepo(provider *data.Provider) *TenantRepo {
	return &TenantRepo{provider: provider}
}

func (r *TenantRepo) db(ctx context.Context) (*gorm.DB, error) {
	// Tenants are global entities, so tenant filter must be skipped
	return r.provider.DB(tenant.SkipFilter(ctx))
}

func (r *TenantRepo) Create(ctx context.Context, t *biz.Tenant) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Create(t).Error
}

func (r *TenantRepo) GetByID(ctx context.Context, id string) (*biz.Tenant, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var t biz.Tenant
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) GetBySlug(ctx context.Context, slug string) (*biz.Tenant, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var t biz.Tenant
	if err := db.Where("slug = ?", slug).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) List(ctx context.Context, q biz.ListQuery) ([]*biz.Tenant, int64, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, 0, err
	}

	tx := db.Model(&biz.Tenant{})
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var items []*biz.Tenant
	if err := tx.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *TenantRepo) Update(ctx context.Context, t *biz.Tenant) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Save(t).Error
}
