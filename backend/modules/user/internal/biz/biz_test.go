package biz_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"saasbase/modules/user/internal/biz"
	"saasbase/platform/tenant"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	if err := tenant.Register(db); err != nil {
		t.Fatalf("failed to register tenant callbacks: %v", err)
	}

	_ = db.Exec("DROP TABLE IF EXISTS user_roles, user_tenants, roles, users CASCADE")

	tables := []any{&biz.User{}, &biz.Role{}, &biz.UserRole{}, &biz.UserTenant{}}
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	return db
}

func TestPhase2_AutoMigrate(t *testing.T) {
	db := setupDB(t)
	if !db.Migrator().HasTable(&biz.User{}) {
		t.Fatal("missing users table")
	}
	if !db.Migrator().HasTable(&biz.Role{}) {
		t.Fatal("missing roles table")
	}
	if !db.Migrator().HasTable(&biz.UserRole{}) {
		t.Fatal("missing user_roles table")
	}
	if !db.Migrator().HasTable(&biz.UserTenant{}) {
		t.Fatal("missing user_tenants table")
	}
}

func TestPhase2_RoleTenantUniqueIndex(t *testing.T) {
	db := setupDB(t)

	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	roleA := &biz.Role{
		ID:             "role-a-admin",
		Name:           "Admin",
		NormalizedName: strings.ToUpper("Admin"),
	}
	if err := db.WithContext(ctxA).Create(roleA).Error; err != nil {
		t.Fatalf("failed to create Admin role in tenant A: %v", err)
	}

	roleB := &biz.Role{
		ID:             "role-b-admin",
		Name:           "Admin",
		NormalizedName: strings.ToUpper("Admin"),
	}
	if err := db.WithContext(ctxB).Create(roleB).Error; err != nil {
		t.Fatalf("failed to create Admin role in tenant B: %v", err)
	}

	// Creating duplicate "Admin" role in tenant A should fail unique constraint
	roleADup := &biz.Role{
		ID:             "role-a-admin-2",
		Name:           "Admin",
		NormalizedName: strings.ToUpper("Admin"),
	}
	if err := db.WithContext(ctxA).Create(roleADup).Error; err == nil {
		t.Fatal("expected duplicate role in tenant A to violate unique index, but succeeded")
	}
}

func TestPhase2_UserTenantCrossTenantRead(t *testing.T) {
	db := setupDB(t)

	u := &biz.User{
		ID:                 "user-1",
		Username:           "testuser",
		NormalizedUsername: "TESTUSER",
		Email:              "user@example.com",
		NormalizedEmail:    "USER@EXAMPLE.COM",
		PasswordHash:       "hashedpassword",
		Status:             biz.UserStatusActive,
	}
	if err := db.WithContext(context.Background()).Create(u).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	utA := &biz.UserTenant{
		UserID:   u.ID,
		TenantID: "tenant-a",
		Status:   biz.MemberStatusActive,
		JoinedAt: time.Now(),
	}
	utB := &biz.UserTenant{
		UserID:   u.ID,
		TenantID: "tenant-b",
		Status:   biz.MemberStatusActive,
		JoinedAt: time.Now(),
	}

	if err := db.WithContext(context.Background()).Create(utA).Error; err != nil {
		t.Fatalf("failed to join tenant A: %v", err)
	}
	if err := db.WithContext(context.Background()).Create(utB).Error; err != nil {
		t.Fatalf("failed to join tenant B: %v", err)
	}

	// Reading user's tenants without tenant filter (cross-tenant)
	var tenants []biz.UserTenant
	skipCtx := tenant.SkipFilter(context.Background())
	if err := db.WithContext(skipCtx).Where("user_id = ?", u.ID).Find(&tenants).Error; err != nil {
		t.Fatalf("failed to read UserTenant records: %v", err)
	}

	if len(tenants) != 2 {
		t.Fatalf("expected 2 tenants for user, got %d", len(tenants))
	}
}
