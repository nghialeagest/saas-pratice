package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	userapi "saasbase/modules/user/api"
	platformdata "saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

func TestAllWriteRoutesEnforceAuthz(t *testing.T) {
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

	_ = db.Exec("DROP TABLE IF EXISTS role_permissions, permissions, sessions, user_roles, user_tenants, roles, users, tasks, notes, tenants CASCADE")
	if err := db.AutoMigrate(module.Tables()...); err != nil {
		t.Fatal(err)
	}

	// Create user "unprivileged" who has an active membership in "tenant-test" but ZERO roles/permissions
	internalSvc, err := module.Resolve[userapi.UserInternalServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve UserInternalServiceServer: %v", err)
	}

	createdUser, err := internalSvc.ProvisionTenantAdmin(bootCtx, &userapi.ProvisionTenantAdminRequest{
		TenantId: "tenant-test",
		Email:    "unprivileged@test.com",
		Username: "unprivileged",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to provision user: %v", err)
	}

	// Strip all roles from this user so they have 0 permissions
	_ = db.Exec("DELETE FROM user_roles WHERE user_id = ?", createdUser.GetId())

	router := newRouter()

	// Login as unprivileged user to obtain valid access token
	loginPayload, _ := json.Marshal(map[string]string{
		"username_or_email": "unprivileged",
		"password":          "Password123!",
		"tenant_id":         "tenant-test",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed with %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.AccessToken

	// Public routes that do not require authz
	isPublic := func(path string) bool {
		return path == "/healthz" || strings.HasPrefix(path, "/v1/auth/")
	}

	// Iterate through all registered routes in router
	testedWriteRoutes := 0
	for _, route := range router.Routes() {
		if isPublic(route.Path) {
			continue
		}

		// Check all mutating methods (POST, PUT, PATCH, DELETE)
		switch route.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			testedWriteRoutes++

			// Build sample path replacing path params like :id
			reqPath := strings.ReplaceAll(route.Path, ":id", "sample-id")

			req := httptest.NewRequest(route.Method, reqPath, bytes.NewBufferString(`{"title":"test"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Tenant-ID", "tenant-test")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Because unprivileged user has no permissions, authz MUST reject with 403 Forbidden!
			if w.Code != http.StatusForbidden {
				t.Errorf("ROUTE MISSING AUTHZ: Route %s %s returned status %d, expected 403 Forbidden! Response: %s",
					route.Method, route.Path, w.Code, w.Body.String())
			}
		}
	}

	if testedWriteRoutes == 0 {
		t.Fatal("no write routes found to scan")
	}

	t.Logf("Successfully scanned %d write routes; all enforce authz with 403 Forbidden on unprivileged access", testedWriteRoutes)
}
