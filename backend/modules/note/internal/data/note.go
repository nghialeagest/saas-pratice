package data

import (
	"context"

	"saasbase/modules/note/internal/biz"
	"saasbase/platform/data"
)

type repo struct {
	provider *data.Provider
}

func NewRepo(provider *data.Provider) biz.Repo {
	return &repo{provider: provider}
}

func (r *repo) Create(ctx context.Context, n *biz.Note) error {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return err
	}
	return db.Create(n).Error
}

func (r *repo) Update(ctx context.Context, n *biz.Note) error {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return err
	}
	return db.Model(&biz.Note{}).Where("id = ?", n.ID).Updates(map[string]any{
		"title":  n.Title,
		"body":   n.Body,
		"status": n.Status,
	}).Error
}

func (r *repo) Get(ctx context.Context, id string) (*biz.Note, error) {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return nil, err
	}
	var n biz.Note
	if err := db.Where("id = ?", id).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *repo) List(ctx context.Context, q biz.ListQuery) ([]*biz.Note, error) {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return nil, err
	}
	tx := db.Model(&biz.Note{})
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.PageSize > 0 {
		tx = tx.Limit(q.PageSize).Offset(max(q.Page-1, 0) * q.PageSize)
	}
	var out []*biz.Note
	if err := tx.Order("created_at desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repo) Count(ctx context.Context, status string) (int64, error) {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return 0, err
	}
	tx := db.Model(&biz.Note{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	var n int64
	err = tx.Count(&n).Error
	return n, err
}
