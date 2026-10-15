package api

import (
	"context"
	"time"

	"saasbase/platform/module"
)

const (
	ResourceTenant = "tenant.tenant"
)

type TenantStatus string

const (
	TenantStatusActive      TenantStatus = "ACTIVE"
	TenantStatusSuspended   TenantStatus = "SUSPENDED"
	TenantStatusDeactivated TenantStatus = "DEACTIVATED"
)

type Tenant struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Slug      string       `json:"slug"`
	Status    TenantStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

type GetTenantRequest struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type UpdateTenantRequest struct {
	ID     string        `json:"id" binding:"required"`
	Name   string        `json:"name"`
	Status *TenantStatus `json:"status"`
}

type ListTenantsRequest struct {
	Status   string `json:"status"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type ListTenantsResponse struct {
	Items []*Tenant `json:"items"`
	Total int64     `json:"total"`
}

// TenantServiceServer is the public interface for tenant management operations.
type TenantServiceServer interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*Tenant, error)
	GetTenant(ctx context.Context, req *GetTenantRequest) (*Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	ListTenants(ctx context.Context, req *ListTenantsRequest) (*ListTenantsResponse, error)
	UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*Tenant, error)
}

func Permissions() []module.Permission {
	return []module.Permission{
		{Namespace: ResourceTenant, Action: "read"},
		{Namespace: ResourceTenant, Action: "write"},
	}
}
