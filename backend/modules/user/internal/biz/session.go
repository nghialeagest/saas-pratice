package biz

import "time"

// Session tracks an active login session in the database.
// This allows revoking tokens even before their JWT expiration.
type Session struct {
	ID           string     `gorm:"type:varchar(64);primaryKey"`
	UserID       string     `gorm:"type:varchar(64);not null;index"`
	TenantID     string     `gorm:"type:varchar(64);not null;index"`
	RefreshToken string     `gorm:"type:varchar(512);uniqueIndex;not null"`
	ExpiresAt    time.Time  `gorm:"not null"`
	RevokedAt    *time.Time
	Username 	 string 
	UserAgent    string     `gorm:"type:varchar(512)"`
	IP           string     `gorm:"type:varchar(45)"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Session) TableName() string { return "sessions" }

func (s *Session) IsActive() bool {
	if s == nil || s.RevokedAt != nil {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}

func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
	s.UpdatedAt = now
}
