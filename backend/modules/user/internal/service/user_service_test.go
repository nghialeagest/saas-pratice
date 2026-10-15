package service_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/modules/user/internal/service"
	"saasbase/platform/authn"
	"saasbase/platform/authz"
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

	_ = db.Exec("DROP TABLE IF EXISTS sessions, user_roles, user_tenants, roles, users CASCADE")

	tables := []any{&biz.User{}, &biz.Role{}, &biz.UserRole{}, &biz.UserTenant{}, &biz.Session{}}
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	return provider
}

func TestGetUser_JSONDoesNotContainPassword(t *testing.T) {
	provider := setupTestDB(t)
	hasher := authn.NewHasher(12)
	userRepo := data.NewUserRepo(provider)
	userRoleRepo := data.NewUserRoleRepo(provider, authz.NewMemoryEventBus())
	roleRepo := data.NewRoleRepo(provider)
	userTenantRepo := data.NewUserTenantRepo(provider)

	svc := service.NewUserService(userRepo, userRoleRepo, roleRepo, userTenantRepo, hasher)

	ctx := context.Background()
	created, err := svc.CreateUser(ctx, &api.CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "SuperSecretPassword123!",
		Name:     "Alice Wonderland",
		Gender:   api.Gender_GENDER_FEMALE,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	got, err := svc.GetUser(ctx, &api.GetUserRequest{Id: created.GetId()})
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	// Marshal protobuf response to JSON
	jsonBytes, err := protojson.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal proto to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)
	lowerJSON := strings.ToLower(jsonStr)

	if strings.Contains(lowerJSON, "password") {
		t.Fatalf("SECURITY VIOLATION: JSON representation of GetUser contains 'password': %s", jsonStr)
	}

	if strings.Contains(lowerJSON, "hash") {
		t.Fatalf("SECURITY VIOLATION: JSON representation of GetUser contains 'hash': %s", jsonStr)
	}

	if strings.Contains(jsonStr, "SuperSecretPassword123!") {
		t.Fatalf("SECURITY VIOLATION: JSON representation contains plain password: %s", jsonStr)
	}
}
