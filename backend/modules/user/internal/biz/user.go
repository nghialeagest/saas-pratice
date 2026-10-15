package biz

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusLocked   UserStatus = "LOCKED"
)

// User is a global entity — NOT tenant-scoped.
// One user account can belong to many tenants via UserTenant.
type User struct {
	ID                 string     `gorm:"type:varchar(64);primaryKey"`
	Username           string     `gorm:"type:varchar(255);not null"`
	NormalizedUsername string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	Email              string     `gorm:"type:varchar(255);not null"`
	NormalizedEmail    string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	Name               string     `gorm:"type:varchar(255)"`
	Gender             string     `gorm:"type:varchar(20)"`
	PasswordHash       string     `gorm:"type:varchar(255);not null"`
	Status             UserStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	AccessFailedCount  int        `gorm:"not null;default:0"`
	LockoutUntil       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (User) TableName() string { return "users" }
