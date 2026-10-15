package tenant_test

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"saasbase/platform/tenant"
)

type RLSTestItem struct {
	ID       string `gorm:"primaryKey;size:64"`
	tenant.Model
	Name     string `gorm:"size:255"`
}

func (RLSTestItem) TableName() string { return "rls_test_items" }

func TestPostgreSQL_RowLevelSecurity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	// 1. Admin/Superuser setup connection
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect as admin: %v", err)
	}

	_ = adminDB.Exec("DROP TABLE IF EXISTS rls_test_items CASCADE")
	if err := adminDB.AutoMigrate(&RLSTestItem{}); err != nil {
		t.Fatalf("failed to migrate rls_test_items: %v", err)
	}

	// 2. Enable RLS on rls_test_items
	if err := tenant.EnableRLS(adminDB, &RLSTestItem{}); err != nil {
		t.Fatalf("failed to enable RLS: %v", err)
	}

	// 3. Create a non-superuser application role without BYPASSRLS
	appRole := "test_app_user"
	appPass := "test_app_pass"
	if err := tenant.SetupAppRole(adminDB, appRole, appPass); err != nil {
		t.Fatalf("failed to setup app role: %v", err)
	}

	// 4. Seed rows for tenant-a and tenant-b using bypass
	_ = tenant.SetDBTenantConfig(adminDB, tenant.SkipFilter(context.Background()))
	_ = adminDB.Create(&RLSTestItem{ID: "item-1", Model: tenant.Model{TenantID: "tenant-a"}, Name: "Item of A"}).Error
	_ = adminDB.Create(&RLSTestItem{ID: "item-2", Model: tenant.Model{TenantID: "tenant-b"}, Name: "Item of B"}).Error

	// 5. Connect as the non-superuser app role (NOBYPASSRLS)
	// Replace user/password in DSN with test_app_user/test_app_pass
	appDSN := "postgres://test_app_user:test_app_pass@localhost:55432/saas?sslmode=disable"
	appDB, err := gorm.Open(postgres.Open(appDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect as app role: %v", err)
	}

	// 6. RAW SQL TEST: Set tenant to tenant-a, execute raw SQL "SELECT * FROM rls_test_items"
	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	if err := tenant.SetDBTenantConfig(appDB, ctxA); err != nil {
		t.Fatalf("failed to set tenant config: %v", err)
	}

	var itemsA []RLSTestItem
	// Notice: RAW SQL with NO WHERE clause!
	if err := appDB.Raw("SELECT * FROM rls_test_items").Scan(&itemsA).Error; err != nil {
		t.Fatalf("raw query failed: %v", err)
	}

	if len(itemsA) != 1 || itemsA[0].ID != "item-1" {
		t.Fatalf("RLS VIOLATION: raw SQL without WHERE clause expected 1 row ('item-1'), got %d rows: %+v", len(itemsA), itemsA)
	}

	// 7. RAW SQL TEST: Set tenant to tenant-b
	ctxB := tenant.NewContext(context.Background(), "tenant-b")
	if err := tenant.SetDBTenantConfig(appDB, ctxB); err != nil {
		t.Fatalf("failed to set tenant config: %v", err)
	}

	var itemsB []RLSTestItem
	if err := appDB.Raw("SELECT * FROM rls_test_items").Scan(&itemsB).Error; err != nil {
		t.Fatalf("raw query failed: %v", err)
	}

	if len(itemsB) != 1 || itemsB[0].ID != "item-2" {
		t.Fatalf("RLS VIOLATION: raw SQL expected item-2 for tenant-b, got: %+v", itemsB)
	}

	// 8. Bypass RLS: SkipFilter allows seeing all rows
	ctxBypass := tenant.SkipFilter(context.Background())
	if err := tenant.SetDBTenantConfig(appDB, ctxBypass); err != nil {
		t.Fatalf("failed to set bypass config: %v", err)
	}

	var itemsAll []RLSTestItem
	if err := appDB.Raw("SELECT * FROM rls_test_items").Scan(&itemsAll).Error; err != nil {
		t.Fatalf("raw query failed: %v", err)
	}

	if len(itemsAll) != 2 {
		t.Fatalf("expected 2 items when bypassing RLS, got %d", len(itemsAll))
	}
}
