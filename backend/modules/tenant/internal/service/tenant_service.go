package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"saasbase/modules/tenant/api"
	"saasbase/modules/tenant/internal/biz"
	"saasbase/platform/errors"
)

type TenantService struct {
	repo biz.Repo
}

var _ api.TenantServiceServer = (*TenantService)(nil)

func NewTenantService(repo biz.Repo) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) CreateTenant(ctx context.Context, req *api.CreateTenantRequest) (*api.Tenant, error) {
	if req.Name == "" {
		return nil, errors.BadRequest("tenant.invalid_argument", "name is required")
	}
	slug := biz.NormalizeSlug(req.Slug)
	if slug == "" {
		return nil, errors.BadRequest("tenant.invalid_argument", "slug is required")
	}

	// Check if slug already exists
	existing, err := s.repo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, errors.BadRequest("tenant.slug_conflict", "tenant slug already in use")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Internal(err)
	}

	t := &biz.Tenant{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Slug:      slug,
		Status:    api.TenantStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, errors.Internal(err)
	}

	return toDTO(t), nil
}

func (s *TenantService) GetTenant(ctx context.Context, req *api.GetTenantRequest) (*api.Tenant, error) {
	if req.ID != "" {
		t, err := s.repo.GetByID(ctx, req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NotFound("tenant.not_found", "tenant not found")
			}
			return nil, errors.Internal(err)
		}
		return toDTO(t), nil
	}

	if req.Slug != "" {
		return s.GetTenantBySlug(ctx, req.Slug)
	}

	return nil, errors.BadRequest("tenant.invalid_argument", "id or slug is required")
}

func (s *TenantService) GetTenantBySlug(ctx context.Context, slug string) (*api.Tenant, error) {
	norm := biz.NormalizeSlug(slug)
	if norm == "" {
		return nil, errors.BadRequest("tenant.invalid_argument", "slug is required")
	}
	t, err := s.repo.GetBySlug(ctx, norm)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("tenant.not_found", "tenant not found")
		}
		return nil, errors.Internal(err)
	}
	return toDTO(t), nil
}

func (s *TenantService) ListTenants(ctx context.Context, req *api.ListTenantsRequest) (*api.ListTenantsResponse, error) {
	q := biz.ListQuery{
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, errors.Internal(err)
	}

	dtos := make([]*api.Tenant, len(items))
	for i, t := range items {
		dtos[i] = toDTO(t)
	}

	return &api.ListTenantsResponse{
		Items: dtos,
		Total: total,
	}, nil
}

func (s *TenantService) UpdateTenant(ctx context.Context, req *api.UpdateTenantRequest) (*api.Tenant, error) {
	if req.ID == "" {
		return nil, errors.BadRequest("tenant.invalid_argument", "id is required")
	}

	t, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("tenant.not_found", "tenant not found")
		}
		return nil, errors.Internal(err)
	}

	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	t.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, errors.Internal(err)
	}

	return toDTO(t), nil
}

func toDTO(t *biz.Tenant) *api.Tenant {
	if t == nil {
		return nil
	}
	return &api.Tenant{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
