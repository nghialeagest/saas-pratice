package biz

import (
	"time"
)

// Role is tenant-scoped. Same role name can exist in different tenants.
// The composite unique index ensures (tenant_id, normalized_name) uniqueness.
type Role struct {
	ID             string `gorm:"type:varchar(64);primaryKey"`
	Name           string `gorm:"type:varchar(255);not null"`
	NormalizedName string `gorm:"type:varchar(255);not null;uniqueIndex:idx_role_tenant_name,priority:2"`
	TenantID       string `gorm:"type:varchar(64);not null;index;uniqueIndex:idx_role_tenant_name,priority:1"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Role) TableName() string { return "roles" }

func (r *Role) TenantField() *string { return &r.TenantID }