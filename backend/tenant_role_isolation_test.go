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
	"saasbase/platform/authz"
	platformdata "saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

// TestRole_DoesNotLeakAcrossTenants tests:
// User U belongs to both tenant A and tenant B.
// U has role Admin (with write permissions) in tenant A.
// U has role Member (with read-only permissions) in tenant B.
// Calling a write API under ctxB (tenant-b) MUST return 403 Forbidden.
// Calling a write API under ctxA (tenant-a) MUST succeed (201 Created).
func TestRole_DoesNotLeakAcrossTenants(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	allTables := module.Tables()
	_, provider := platformdata.IsolatedTestDB(t, dsn, allTables...)

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatalf("failed to init modules: %v", err)
	}

	bootCtx := tenant.SkipFilter(tenant.NewContext(context.Background(), ""))

	internalSvc, err := module.Resolve[userapi.UserInternalServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve UserInternalServiceServer: %v", err)
	}
	roleSvc, err := module.Resolve[userapi.RoleServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve RoleServiceServer: %v", err)
	}
	userSvc, err := module.Resolve[userapi.UserServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve UserServiceServer: %v", err)
	}
	permMgr, err := module.Resolve[userapi.PermissionManager]()
	if err != nil {
		t.Fatalf("failed to resolve PermissionManager: %v", err)
	}

	// 1. Provision user U in tenant-a as Admin
	userProto, err := internalSvc.ProvisionTenantAdmin(bootCtx, &userapi.ProvisionTenantAdminRequest{
		TenantId: "tenant-a",
		Email:    "alice.multi@company.com",
		Username: "alicemulti",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to provision admin in tenant-a: %v", err)
	}
	userID := userProto.GetId()

	// 2. Add user U to tenant-b as regular Member
	_, err = internalSvc.JoinTenant(bootCtx, &userapi.JoinTenantRequest{
		UserId:   userID,
		TenantId: "tenant-b",
	})
	if err != nil {
		t.Fatalf("failed to join tenant-b: %v", err)
	}

	// Create "Member" role in tenant-b
	ctxB := tenant.NewContext(context.Background(), "tenant-b")
	memberRole, err := roleSvc.CreateRole(ctxB, &userapi.CreateRoleRequest{
		Name: "Member",
	})
	if err != nil {
		t.Fatalf("failed to create member role in tenant-b: %v", err)
	}

	// Assign U to Member role in tenant-b
	_, err = userSvc.AssignRoles(ctxB, &userapi.AssignRolesRequest{
		UserId:  userID,
		RoleIds: []string{memberRole.GetId()},
	})
	if err != nil {
		t.Fatalf("failed to assign member role in tenant-b: %v", err)
	}

	// Grant only read permission to "Member" role in tenant-b
	err = permMgr.AssignPermissions(ctxB, memberRole.GetId(), "tenant-b", []authz.Permission{
		{Namespace: "note.note", Action: "read"},
	})
	if err != nil {
		t.Fatalf("failed to assign read permission to member role: %v", err)
	}

	// 3. Login to both tenants to obtain access tokens
	router := newRouter()

	loginA := func() string {
		body, _ := json.Marshal(map[string]string{
			"username_or_email": "alicemulti",
			"password":          "Password123!",
			"tenant_id":         "tenant-a",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("login tenant-a failed: %d, body: %s", w.Code, w.Body.String())
		}
		var resp struct {
			AccessToken string `json:"access_token"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.AccessToken
	}

	loginB := func() string {
		body, _ := json.Marshal(map[string]string{
			"username_or_email": "alicemulti",
			"password":          "Password123!",
			"tenant_id":         "tenant-b",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("login tenant-b failed: %d, body: %s", w.Code, w.Body.String())
		}
		var resp struct {
			AccessToken string `json:"access_token"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.AccessToken
	}

	tokenA := loginA()
	tokenB := loginB()

	// 4. CRITICAL ASSERTION:
	// U calls write API (POST /v1/notes) under tenant B -> MUST BE 403 FORBIDDEN!
	// (Admin role from tenant A must NOT leak to tenant B)
	noteBody := `{"title":"Attempted Note","body":"Should fail in tenant B"}`
	reqWriteB := httptest.NewRequest(http.MethodPost, "/v1/notes", bytes.NewReader([]byte(noteBody)))
	reqWriteB.Header.Set("Content-Type", "application/json")
	reqWriteB.Header.Set("Authorization", "Bearer "+tokenB)
	reqWriteB.Header.Set(tenant.Header, "tenant-b")
	wWriteB := httptest.NewRecorder()
	router.ServeHTTP(wWriteB, reqWriteB)

	if wWriteB.Code != http.StatusForbidden {
		t.Fatalf("SECURITY LEAK: User with Member role in tenant-b called write API and got %d, expected 403 Forbidden! Body: %s",
			wWriteB.Code, wWriteB.Body.String())
	}

	// 5. User U calls read API (GET /v1/notes) under tenant B -> MUST BE 200 OK
	reqReadB := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	reqReadB.Header.Set("Authorization", "Bearer "+tokenB)
	reqReadB.Header.Set(tenant.Header, "tenant-b")
	wReadB := httptest.NewRecorder()
	router.ServeHTTP(wReadB, reqReadB)

	if wReadB.Code != http.StatusOK {
		t.Fatalf("User with Member role in tenant-b called read API and got %d, expected 200 OK! Body: %s",
			wReadB.Code, wReadB.Body.String())
	}

	// 6. User U calls write API (POST /v1/notes) under tenant A -> MUST BE 201 CREATED
	// (Admin role is active in tenant A)
	reqWriteA := httptest.NewRequest(http.MethodPost, "/v1/notes", bytes.NewReader([]byte(noteBody)))
	reqWriteA.Header.Set("Content-Type", "application/json")
	reqWriteA.Header.Set("Authorization", "Bearer "+tokenA)
	reqWriteA.Header.Set(tenant.Header, "tenant-a")
	wWriteA := httptest.NewRecorder()
	router.ServeHTTP(wWriteA, reqWriteA)

	if wWriteA.Code != http.StatusCreated {
		t.Fatalf("User with Admin role in tenant-a called write API and got %d, expected 201 Created! Body: %s",
			wWriteA.Code, wWriteA.Body.String())
	}
}
