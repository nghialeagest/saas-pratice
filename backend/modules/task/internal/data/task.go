package data

import (
	"context"

	"saasbase/modules/task/internal/biz"
	"saasbase/platform/data"
)

type repo struct {
	provider *data.Provider
}

func NewRepo(provider *data.Provider) biz.Repo {
	return &repo{provider: provider}
}

func (r *repo) Create(ctx context.Context, t *biz.Task) error {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return err
	}
	return db.Create(t).Error
}

func (r *repo) Get(ctx context.Context, id string) (*biz.Task, error) {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return nil, err
	}
	var t biz.Task
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repo) Count(ctx context.Context) (int64, error) {
	db, err := r.provider.DB(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	err = db.Model(&biz.Task{}).Count(&n).Error
	return n, err
}
