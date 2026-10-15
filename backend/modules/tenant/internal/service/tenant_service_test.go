package service_test

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"saasbase/modules/tenant/api"
	"saasbase/modules/tenant/internal/biz"
	"saasbase/modules/tenant/internal/data"
	"saasbase/modules/tenant/internal/service"
	platformdata "saasbase/platform/data"
	"saasbase/platform/tenant"
)

func setupTestDB(t *testing.T) *platformdata.Provider {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	provider := platformdata.NewProvider(platformdata.NewSharedConn(dsn), nil)
	t.Cleanup(func() { _ = provider.Close() })

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	_ = tenant.Register(db)

	_ = db.Exec("DROP TABLE IF EXISTS tenants CASCADE")
	if err := db.AutoMigrate(&biz.Tenant{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	return provider
}

func TestTenantService_CRUD(t *testing.T) {
	provider := setupTestDB(t)
	repo := data.NewTenantRepo(provider)
	svc := service.NewTenantService(repo)

	ctx := context.Background()

	// 1. Create tenant
	created, err := svc.CreateTenant(ctx, &api.CreateTenantRequest{
		Name: "Acme Corporation",
		Slug: "ACME-CORP",
	})
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}
	if created.Slug != "acme-corp" {
		t.Fatalf("expected normalized slug 'acme-corp', got '%s'", created.Slug)
	}
	if created.Status != api.TenantStatusActive {
		t.Fatalf("expected ACTIVE status, got %s", created.Status)
	}

	// 2. Duplicate slug must fail
	_, err = svc.CreateTenant(ctx, &api.CreateTenantRequest{
		Name: "Acme Duplicate",
		Slug: "acme-corp",
	})
	if err == nil {
		t.Fatal("expected duplicate slug to fail, got nil error")
	}

	// 3. Get by ID
	gotByID, err := svc.GetTenant(ctx, &api.GetTenantRequest{ID: created.ID})
	if err != nil {
		t.Fatalf("failed to get tenant by id: %v", err)
	}
	if gotByID.Name != "Acme Corporation" {
		t.Fatalf("unexpected name: %s", gotByID.Name)
	}

	// 4. Get by Slug
	gotBySlug, err := svc.GetTenantBySlug(ctx, "acme-corp")
	if err != nil {
		t.Fatalf("failed to get tenant by slug: %v", err)
	}
	if gotBySlug.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, gotBySlug.ID)
	}

	// 5. Update tenant
	suspended := api.TenantStatusSuspended
	updated, err := svc.UpdateTenant(ctx, &api.UpdateTenantRequest{
		ID:     created.ID,
		Name:   "Acme Global",
		Status: &suspended,
	})
	if err != nil {
		t.Fatalf("failed to update tenant: %v", err)
	}
	if updated.Name != "Acme Global" || updated.Status != api.TenantStatusSuspended {
		t.Fatalf("unexpected updated fields: name=%s status=%s", updated.Name, updated.Status)
	}

	// 6. List tenants
	listResp, err := svc.ListTenants(ctx, &api.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("failed to list tenants: %v", err)
	}
	if listResp.Total < 1 || len(listResp.Items) < 1 {
		t.Fatalf("expected at least 1 tenant in list, got total=%d items=%d", listResp.Total, len(listResp.Items))
	}
}
