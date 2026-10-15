package data

import (
	"context"
	"time"

	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/data"
	"saasbase/platform/tenant"
)

type SessionRepo struct {
	provider *data.Provider
}

func NewSessionRepo(provider *data.Provider) *SessionRepo {
	return &SessionRepo{provider: provider}
}

func (r *SessionRepo) db(ctx context.Context) (*gorm.DB, error) {
	// Sessions are looked up by session ID / token globally
	return r.provider.DB(tenant.SkipFilter(ctx))
}

func (r *SessionRepo) Create(ctx context.Context, s *biz.Session) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Create(s).Error
}

func (r *SessionRepo) Get(ctx context.Context, id string) (*biz.Session, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var s biz.Session
	if err := db.Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) FindByRefreshToken(ctx context.Context, token string) (*biz.Session, error) {
	db, err := r.db(ctx)
	if err != nil {
		return nil, err
	}
	var s biz.Session
	if err := db.Where("refresh_token = ?", token).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, id string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	return db.Model(&biz.Session{}).Where("id = ?", id).Updates(map[string]any{
		"revoked_at": &now,
		"updated_at": now,
	}).Error
}

func (r *SessionRepo) RevokeByUser(ctx context.Context, userID string) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	return db.Model(&biz.Session{}).Where("user_id = ? AND revoked_at IS NULL", userID).Updates(map[string]any{
		"revoked_at": &now,
		"updated_at": now,
	}).Error
}

func (r *SessionRepo) Update(ctx context.Context, s *biz.Session) error {
	db, err := r.db(ctx)
	if err != nil {
		return err
	}
	return db.Model(&biz.Session{}).Where("id = ?", s.ID).Updates(map[string]any{
		"refresh_token": s.RefreshToken,
		"expires_at":    s.ExpiresAt,
		"revoked_at":    s.RevokedAt,
		"user_agent":    s.UserAgent,
		"ip":            s.IP,
		"updated_at":    s.UpdatedAt,
	}).Error
}
