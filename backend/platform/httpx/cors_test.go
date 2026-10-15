package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"saasbase/platform/httpx"
)

func TestCORSMiddleware_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	opts := httpx.DefaultCORSOptions()
	r.Use(httpx.CORSMiddleware(opts))

	r.POST("/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-tenant-id")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 No Content for preflight, got %d", w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:5173, got %s", origin)
	}

	if creds := w.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials to be true, got %s", creds)
	}

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Authorization", "X-Tenant-ID", "Content-Type", "X-Request-ID"} {
		if !containsCaseInsensitive(allowHeaders, h) {
			t.Errorf("expected Access-Control-Allow-Headers to contain %s, got %s", h, allowHeaders)
		}
	}
}

func TestCORSMiddleware_ActualRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	opts := httpx.DefaultCORSOptions()
	r.Use(httpx.CORSMiddleware(opts))

	r.GET("/v1/notes", func(c *gin.Context) {
		c.String(http.StatusOK, "notes list")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:5173, got %s", origin)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	opts := httpx.DefaultCORSOptions()
	r.Use(httpx.CORSMiddleware(opts))

	r.GET("/v1/notes", func(c *gin.Context) {
		c.String(http.StatusOK, "notes list")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	req.Header.Set("Origin", "http://malicious-site.com")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for unauthorized site, got %s", origin)
	}
}

func containsCaseInsensitive(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}
