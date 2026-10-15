package biz

import (
	"context"
	"time"

	"saasbase/platform/tenant"
)

const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
)

type Note struct {
	ID        string `gorm:"primaryKey;size:36"`
	tenant.Model
	Title     string `gorm:"size:255;not null"`
	Body      string
	Status    string `gorm:"size:32;index;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Note) TableName() string { return "notes" }

type ListQuery struct {
	Status   string
	Page     int
	PageSize int
}

type Repo interface {
	Create(ctx context.Context, n *Note) error
	Update(ctx context.Context, n *Note) error
	Get(ctx context.Context, id string) (*Note, error)
	List(ctx context.Context, q ListQuery) ([]*Note, error)
	Count(ctx context.Context, status string) (int64, error)
}

func (n *Note) Publish() { n.Status = StatusPublished }
