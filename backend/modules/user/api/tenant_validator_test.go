package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"saasbase/modules/user/api"
	"saasbase/platform/authn"
	"saasbase/platform/tenant"
)

type mockInternalService struct {
	api.UnimplementedUserInternalServiceServer
	memberships map[string]bool // key: "userID:tenantID" -> true if active
}

func (m *mockInternalService) CheckUserTenant(ctx context.Context, req *api.CheckUserTenantRequest) (*api.CheckUserTenantResponse, error) {
	key := req.GetUserId() + ":" + req.GetTenantId()
	allowed := m.memberships[key]
	return &api.CheckUserTenantResponse{Allowed: allowed}, nil
}

func TestTenantValidator_CrossTenantForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockInternal := &mockInternalService{
		memberships: map[string]bool{
			"user-alice:tenant-a": true, // Alice only belongs to tenant-a
			"user-bob:tenant-b":   true, // Bob belongs to tenant-b
		},
	}

	validator := api.NewTenantValidator(mockInternal)

	cfg := authn.DefaultTokenConfig()

	r := gin.New()
	r.Use(authn.GinMiddleware(authn.MiddlewareOptions{
		TokenConfig: cfg,
	}))
	r.Use(tenant.Middleware(tenant.Options{
		Validator: validator,
	}))

	r.GET("/v1/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tenant": tenant.FromContext(c.Request.Context())})
	})

	// Alice has a valid token
	aliceToken, err := authn.IssueAccess(cfg, "user-alice", "tenant-a", "sess-1")
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	// 1. CRITICAL TEST: Alice attempts to access tenant-b by sending X-Tenant-ID: tenant-b -> MUST BE 403 FORBIDDEN
	reqSpoofed := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
	reqSpoofed.Header.Set("Authorization", "Bearer "+aliceToken)
	reqSpoofed.Header.Set("X-Tenant-ID", "tenant-b")
	wSpoofed := httptest.NewRecorder()
	r.ServeHTTP(wSpoofed, reqSpoofed)

	if wSpoofed.Code != http.StatusForbidden {
		t.Fatalf("CRITICAL SECURITY HOLE: User from tenant-a requested tenant-b and got %d, WANT 403 Forbidden", wSpoofed.Code)
	}

	// 2. Alice requests her legitimate tenant-a -> MUST BE 200 OK
	reqLegit := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
	reqLegit.Header.Set("Authorization", "Bearer "+aliceToken)
	reqLegit.Header.Set("X-Tenant-ID", "tenant-a")
	wLegit := httptest.NewRecorder()
	r.ServeHTTP(wLegit, reqLegit)

	if wLegit.Code != http.StatusOK {
		t.Fatalf("legitimate tenant request failed: %d", wLegit.Code)
	}
}
