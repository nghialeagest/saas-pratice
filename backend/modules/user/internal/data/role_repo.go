package data

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/data"
)

type RoleRepo struct {
	provider *data.Provider
}

func NewRoleRepo(provider *data.Provider) *RoleRepo {
	return &RoleRepo{provider: provider}
}

func (r *RoleRepo) db(ctx context.Context) (*gorm.DB, error) {
	return r.provider.DB(ctx)
}

func (r *RoleRepo) Create(ctx context.Context, role *biz.Role) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	if role.NormalizedName == "" && role.Name != "" {
		role.NormalizedName = strings.ToUpper(role.Name)
	}
	return db.Create(role).Error
}

func (r *RoleRepo) Get(ctx context.Context, id string) (*biz.Role, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var role biz.Role
	if err := db.Where("id = ?", id).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepo) FindByName(ctx context.Context, name string) (*biz.Role, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var role biz.Role
	if err := db.Where("normalized_name = ?", strings.ToUpper(name)).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepo) List(ctx context.Context) ([]*biz.Role, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*biz.Role
	if err := db.Order("created_at asc").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepo) FindByIDs(ctx context.Context, ids []string) ([]*biz.Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*biz.Role
	if err := db.Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepo) Delete(ctx context.Context, id string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Where("id = ?", id).Delete(&biz.Role{}).Error
}
