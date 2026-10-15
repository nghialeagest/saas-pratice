package data

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/data"
	"saasbase/platform/tenant"
)

type UserRepo struct {
	provider *data.Provider
}

func NewUserRepo(provider *data.Provider) *UserRepo {
	return &UserRepo{provider: provider}
}

func (r *UserRepo) db(ctx context.Context) (*gorm.DB, error) {
	// User is a global entity spanning all tenants, so skip tenant filter
	return r.provider.DB(tenant.SkipFilter(ctx))
}

func (r *UserRepo) Create(ctx context.Context, u *biz.User) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Create(u).Error
}

func (r *UserRepo) Update(ctx context.Context, u *biz.User) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Model(&biz.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"name":                u.Name,
		"gender":              u.Gender,
		"status":              u.Status,
		"access_failed_count": u.AccessFailedCount,
		"lockout_until":       u.LockoutUntil,
		"password_hash":       u.PasswordHash,
		"updated_at":          u.UpdatedAt,
	}).Error
}

func (r *UserRepo) Get(ctx context.Context, id string) (*biz.User, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var u biz.User
	if err := db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByNormalizedUsername(ctx context.Context, normUsername string) (*biz.User, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var u biz.User
	if err := db.Where("normalized_username = ?", normUsername).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByNormalizedEmail(ctx context.Context, normEmail string) (*biz.User, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var u biz.User
	if err := db.Where("normalized_email = ?", normEmail).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByUsernameOrEmail(ctx context.Context, identifier string) (*biz.User, error) {
	normalized := strings.ToUpper(identifier)
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var u biz.User
	if err := db.Where("normalized_username = ? OR normalized_email = ?", normalized, normalized).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context, page, pageSize int, search string) ([]*biz.User, int64, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, 0, err
	}
	tx := db.Model(&biz.User{})

	// users is a global table, so membership is what scopes a listing to the
	// current tenant. Without this join a tenant admin would see every account
	// in the system, including users of other tenants.
	if tid := tenant.FromContext(ctx); tid != "" {
		tx = tx.Joins("JOIN user_tenants ON user_tenants.user_id = users.id").
			Where("user_tenants.tenant_id = ? AND user_tenants.status = ?", tid, string(biz.MemberStatusActive))
	}

	if search != "" {
		s := "%" + strings.ToUpper(search) + "%"
		tx = tx.Where("normalized_username LIKE ? OR normalized_email LIKE ?", s, s)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageSize > 0 {
		tx = tx.Limit(pageSize).Offset(max(page-1, 0) * pageSize)
	}
	var users []*biz.User
	if err := tx.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Where("id = ?", id).Delete(&biz.User{}).Error
}
