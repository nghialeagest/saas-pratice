package biz

import "time"

// Permission represents a tenant-scoped permission registered in the system.
type Permission struct {
	ID        string    `gorm:"type:varchar(64);primaryKey"`
	TenantID  string    `gorm:"type:varchar(64);not null;index;uniqueIndex:idx_perm_tenant_ns_act,priority:1"`
	Namespace string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_tenant_ns_act,priority:2"`
	Action    string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_tenant_ns_act,priority:3"`
	CreatedAt time.Time
}

func (Permission) TableName() string { return "permissions" }

// RolePermission links a tenant-scoped role to granted permissions.
type RolePermission struct {
	RoleID    string    `gorm:"type:varchar(64);primaryKey"`
	TenantID  string    `gorm:"type:varchar(64);primaryKey"`
	Namespace string    `gorm:"type:varchar(64);primaryKey"`
	Action    string    `gorm:"type:varchar(64);primaryKey"`
	CreatedAt time.Time
}

func (RolePermission) TableName() string { return "role_permissions" }
