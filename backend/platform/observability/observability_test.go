package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"saasbase/platform/authn"
	"saasbase/platform/observability"
)

func TestLogging_ContextAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(buf, 0) // LevelDebug / 0

	ctx := context.Background()
	ctx = observability.NewRequestContext(ctx, "req-12345")
	ctx = authn.NewContext(ctx, authn.Identity{UserID: "usr-99", TenantID: "tenant-xyz"})

	logger.InfoContext(ctx, "test event", "action", "create_order")

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log json: %v, raw: %s", err, buf.String())
	}

	if logEntry["request_id"] != "req-12345" {
		t.Errorf("expected request_id req-12345, got %v", logEntry["request_id"])
	}
	if logEntry["user_id"] != "usr-99" {
		t.Errorf("expected user_id usr-99, got %v", logEntry["user_id"])
	}
	if logEntry["tenant_id"] != "tenant-xyz" {
		t.Errorf("expected tenant_id tenant-xyz, got %v", logEntry["tenant_id"])
	}
}

func TestLogging_SecretSanitization(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := observability.NewLogger(buf, 0)

	ctx := context.Background()

	// Try logging sensitive fields directly
	rawSecretPassword := "MyP@ssw0rd123!"
	rawJWTToken := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.dummy_token"
	rawBcryptHash := "$2a$12$e8k8yO.u0kR3k2W1Q2h6g.5ZkO6GvHwI5aM1x7Z6jY3s5uL9w6M7q"
	rawAuthHeader := "Bearer eyJhbGciOiJIUzUxMi..."

	logger.InfoContext(ctx, "incoming credentials",
		"password", rawSecretPassword,
		"token", rawJWTToken,
		"hash", rawBcryptHash,
		"authorization", rawAuthHeader,
		"refresh_token", "xyz-refresh-123",
	)

	logOutput := buf.String()

	// ASSERTION: Secrets must NEVER appear in the log output!
	if strings.Contains(logOutput, rawSecretPassword) {
		t.Fatalf("SECURITY VIOLATION: raw password leaked into log: %s", logOutput)
	}
	if strings.Contains(logOutput, rawJWTToken) {
		t.Fatalf("SECURITY VIOLATION: raw token leaked into log: %s", logOutput)
	}
	if strings.Contains(logOutput, rawBcryptHash) {
		t.Fatalf("SECURITY VIOLATION: bcrypt hash leaked into log: %s", logOutput)
	}
	if strings.Contains(logOutput, rawAuthHeader) {
		t.Fatalf("SECURITY VIOLATION: Authorization header leaked into log: %s", logOutput)
	}
	if strings.Contains(logOutput, "xyz-refresh-123") {
		t.Fatalf("SECURITY VIOLATION: refresh_token leaked into log: %s", logOutput)
	}

	// Verify that they were replaced with REDACTED
	if !strings.Contains(logOutput, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in sanitized log, got: %s", logOutput)
	}
}

func TestRequestID_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(observability.RequestIDMiddleware())

	var capturedReqID string
	r.GET("/ping", func(c *gin.Context) {
		capturedReqID = observability.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// 1. Without incoming X-Request-ID -> should generate one
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respHeaderID := w.Header().Get(observability.HeaderRequestID)
	if respHeaderID == "" {
		t.Fatal("expected X-Request-ID on response header")
	}
	if capturedReqID != respHeaderID {
		t.Fatalf("context request_id (%s) differs from header (%s)", capturedReqID, respHeaderID)
	}

	// 2. With incoming X-Request-ID -> should propagate it
	reqWithID := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqWithID.Header.Set(observability.HeaderRequestID, "custom-trace-id-123")
	wWithID := httptest.NewRecorder()
	r.ServeHTTP(wWithID, reqWithID)

	if wWithID.Header().Get(observability.HeaderRequestID) != "custom-trace-id-123" {
		t.Fatalf("expected propagated ID, got %s", wWithID.Header().Get(observability.HeaderRequestID))
	}
	if capturedReqID != "custom-trace-id-123" {
		t.Fatalf("expected captured ID custom-trace-id-123, got %s", capturedReqID)
	}
}

func TestTracing_InitAndHTTPMiddleware(t *testing.T) {
	_, shutdown, err := observability.InitTracer("test-service", nil)
	if err != nil {
		t.Fatalf("failed to init tracer: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(observability.HTTPTraceMiddleware())
	r.GET("/orders/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"order": c.Param("id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/orders/101", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
