package biz

import (
	"time"
)

// UserRole links a user to a role within a specific tenant.
// tenant_id is part of the composite primary key.
type UserRole struct {
	UserID    string    `gorm:"type:varchar(64);primaryKey"`
	RoleID    string    `gorm:"type:varchar(64);primaryKey"`
	TenantID  string    `gorm:"type:varchar(64);primaryKey"`
	CreatedAt time.Time
}

func (UserRole) TableName() string { return "user_roles" }

func (ur *UserRole) TenantField() *string { return &ur.TenantID }
