package biz

import (
	"context"
	"strings"
	"time"

	"saasbase/modules/tenant/api"
)

// Tenant is a global entity representing a tenant (organization/workspace).
// It deliberately does NOT embed tenant.Model because tenant records are global.
type Tenant struct {
	ID        string           `gorm:"type:varchar(64);primaryKey"`
	Name      string           `gorm:"type:varchar(255);not null"`
	Slug      string           `gorm:"type:varchar(64);not null;uniqueIndex"`
	Status    api.TenantStatus `gorm:"type:varchar(32);not null;default:'ACTIVE'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Tenant) TableName() string { return "tenants" }

func NormalizeSlug(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

type ListQuery struct {
	Status   string
	Page     int
	PageSize int
}

type Repo interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	List(ctx context.Context, q ListQuery) ([]*Tenant, int64, error)
	Update(ctx context.Context, t *Tenant) error
}
