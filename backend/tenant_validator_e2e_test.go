package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	userapi "saasbase/modules/user/api"
	platformdata "saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

func TestE2E_TenantValidator_BlocksCrossTenantAttack(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	provider := platformdata.NewProvider(platformdata.NewSharedConn(dsn), nil)
	t.Cleanup(func() { _ = provider.Close() })

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatal(err)
	}

	bootCtx := tenant.SkipFilter(tenant.NewContext(context.Background(), ""))
	db, err := provider.DB(bootCtx)
	if err != nil {
		t.Fatal(err)
	}

	_ = db.Exec("DROP TABLE IF EXISTS sessions, user_roles, user_tenants, roles, users, tasks, notes CASCADE")
	if err := db.AutoMigrate(module.Tables()...); err != nil {
		t.Fatal(err)
	}

	// Resolve UserInternalServiceServer to provision Alice in tenant-a
	internalSvc, err := module.Resolve[userapi.UserInternalServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve UserInternalServiceServer: %v", err)
	}

	_, err = internalSvc.ProvisionTenantAdmin(bootCtx, &userapi.ProvisionTenantAdminRequest{
		TenantId: "tenant-a",
		Email:    "alice@company.com",
		Username: "alice",
		Password: "AlicePassword123!",
	})
	if err != nil {
		t.Fatalf("failed to provision alice: %v", err)
	}

	// Initialize the real router using the full middleware pipeline
	router := newRouter()

	// 1. Alice logs in via POST /v1/auth/login
	loginPayload, _ := json.Marshal(map[string]string{
		"username_or_email": "alice",
		"password":          "AlicePassword123!",
		"tenant_id":         "tenant-a",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if loginResp.AccessToken == "" {
		t.Fatal("empty access token returned")
	}

	// 2. CRITICAL TEST: Alice attempts to send X-Tenant-ID: tenant-b
	// Alice only belongs to tenant-a. This cross-tenant access MUST return 403 Forbidden!
	reqAttack := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	reqAttack.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	reqAttack.Header.Set("X-Tenant-ID", "tenant-b")
	wAttack := httptest.NewRecorder()
	router.ServeHTTP(wAttack, reqAttack)

	if wAttack.Code != http.StatusForbidden {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY: User from tenant-a requested tenant-b and got %d, expected 403 Forbidden! Body: %s",
			wAttack.Code, wAttack.Body.String())
	}

	// 3. Legitimate request: Alice requests her own tenant-a -> MUST return 200 OK
	reqLegit := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	reqLegit.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	reqLegit.Header.Set("X-Tenant-ID", "tenant-a")
	wLegit := httptest.NewRecorder()
	router.ServeHTTP(wLegit, reqLegit)

	if wLegit.Code != http.StatusOK {
		t.Fatalf("legitimate request to tenant-a failed with %d: %s", wLegit.Code, wLegit.Body.String())
	}
}
