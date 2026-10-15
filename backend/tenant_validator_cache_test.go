package main

import (
	"context"
	"os"
	"testing"
	"time"

	userapi "saasbase/modules/user/api"
	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

// TestTenantValidator_CacheLifecycle tests:
// 1. Within TTL: Validation passes from cache even if DB membership is removed out-of-band.
// 2. After TTL: Cache expires, queries DB, and blocks the removed user.
// 3. API Removal: Calling member removal API immediately invalidates the cache so subsequent validation fails immediately without waiting for TTL.
func TestTenantValidator_CacheLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	allTables := module.Tables()
	db, provider := data.IsolatedTestDB(t, dsn, allTables...)

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatalf("failed to init modules: %v", err)
	}

	bootCtx := tenant.SkipFilter(tenant.NewContext(context.Background(), ""))

	internalSvc, err := module.Resolve[userapi.UserInternalServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve UserInternalServiceServer: %v", err)
	}
	eventBus, err := module.Resolve[authz.EventBus]()
	if err != nil {
		t.Fatalf("failed to resolve EventBus: %v", err)
	}

	// 1. Provision user U in tenant-a
	u, err := internalSvc.ProvisionTenantAdmin(bootCtx, &userapi.ProvisionTenantAdminRequest{
		TenantId: "tenant-a",
		Email:    "cache.user@company.com",
		Username: "cacheuser",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to provision user: %v", err)
	}

	// 2. Setup TenantValidator with a 150ms TTL for testing
	ttl := 150 * time.Millisecond
	validator := userapi.NewTenantValidatorWithTTL(internalSvc, ttl)
	validator.SubscribeToEventBus(eventBus)

	ctxA := authn.NewContext(context.Background(), authn.Identity{
		UserID:   u.GetId(),
		TenantID: "tenant-a",
	})

	// Initial validation MUST succeed and prime the cache
	if err := validator.Validate(ctxA, "tenant-a"); err != nil {
		t.Fatalf("initial validation should pass: %v", err)
	}

	// PART 1: Remove membership out-of-band directly in DB (simulates DB change without event)
	if err := db.Exec("DELETE FROM user_tenants WHERE user_id = ? AND tenant_id = ?", u.GetId(), "tenant-a").Error; err != nil {
		t.Fatalf("failed to delete user_tenant in DB: %v", err)
	}

	// ASSERTION 1: Within TTL, validation still passes (acceptable, intentional cache design)
	if err := validator.Validate(ctxA, "tenant-a"); err != nil {
		t.Fatalf("expected validation to pass within TTL from cache, got: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(200 * time.Millisecond)

	// ASSERTION 2: After TTL, validation re-checks DB and MUST be blocked
	if err := validator.Validate(ctxA, "tenant-a"); err == nil {
		t.Fatalf("SECURITY LEAK: user still validated for tenant-a after TTL expired!")
	}

	// PART 2: API Removal MUST invalidate cache immediately
	// Add user to tenant-b
	_, err = internalSvc.JoinTenant(bootCtx, &userapi.JoinTenantRequest{
		UserId:   u.GetId(),
		TenantId: "tenant-b",
	})
	if err != nil {
		t.Fatalf("failed to join tenant-b: %v", err)
	}

	ctxB := authn.NewContext(context.Background(), authn.Identity{
		UserID:   u.GetId(),
		TenantID: "tenant-b",
	})

	// Prime cache for tenant-b
	if err := validator.Validate(ctxB, "tenant-b"); err != nil {
		t.Fatalf("validation for tenant-b should pass: %v", err)
	}

	// Remove member via API (which publishes EventUserLeftTenant)
	type removableService interface {
		RemoveUserTenant(ctx context.Context, userID, tenantID string) error
	}
	remSvc, ok := internalSvc.(removableService)
	if !ok {
		t.Fatal("internalSvc does not implement RemoveUserTenant")
	}

	if err := remSvc.RemoveUserTenant(bootCtx, u.GetId(), "tenant-b"); err != nil {
		t.Fatalf("failed to remove user via API: %v", err)
	}

	// ASSERTION 3: Calling Validate immediately (without waiting for TTL) MUST fail immediately!
	if err := validator.Validate(ctxB, "tenant-b"); err == nil {
		t.Fatalf("SECURITY LEAK: user still validated for tenant-b immediately after API removal! Cache was not invalidated.")
	}
}
