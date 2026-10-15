package biz

import (
	"context"
	"time"

	"saasbase/platform/tenant"
)

type Task struct {
	ID        string `gorm:"primaryKey;size:36"`
	tenant.Model
	Title     string `gorm:"size:255;not null"`
	NoteID    string `gorm:"size:36;index"`
	NoteTitle string `gorm:"size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Task) TableName() string { return "tasks" }

type Repo interface {
	Create(ctx context.Context, t *Task) error
	Get(ctx context.Context, id string) (*Task, error)
	Count(ctx context.Context) (int64, error)
}
