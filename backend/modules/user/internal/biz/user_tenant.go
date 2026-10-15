package biz

import "time"

type MemberStatus string

const (
	MemberStatusActive   MemberStatus = "ACTIVE"
	MemberStatusInactive MemberStatus = "INACTIVE"
)

// UserTenant records which tenants a user belongs to.
// It does NOT embed tenant.Model because:
// - The tenant callback would auto-filter by current tenant
// - But this table must be queryable across tenants to determine
//   which tenants a user belongs to
// - tenant_id is a plain PK column, not from embedded Model
type UserTenant struct {
	UserID    string       `gorm:"type:varchar(64);primaryKey"`
	TenantID  string       `gorm:"type:varchar(64);primaryKey"`
	Status    MemberStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	JoinedAt  time.Time    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserTenant) TableName() string { return "user_tenants" }
