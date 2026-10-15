package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/handler"
	"saasbase/platform/errors"
)

type mockAuthService struct {
	api.UnimplementedAuthServiceServer
}

func (m *mockAuthService) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	if req.UsernameOrEmail == "admin" && req.Password == "password123" {
		return &api.LoginResponse{
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-refresh-token",
			ExpiresIn:    900,
		}, nil
	}
	return nil, errors.Unauthorized("invalid username, email or password")
}

func (m *mockAuthService) Refresh(ctx context.Context, req *api.RefreshRequest) (*api.RefreshResponse, error) {
	if req.RefreshToken == "mock-refresh-token" {
		return &api.RefreshResponse{
			AccessToken:  "mock-new-access-token",
			RefreshToken: "mock-new-refresh-token",
			ExpiresIn:    900,
		}, nil
	}
	return nil, errors.Unauthorized("invalid or revoked refresh token")
}

func (m *mockAuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	return &api.LogoutResponse{}, nil
}

func TestAuthHandler_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewAuthHandler(&mockAuthService{})

	r := gin.New()
	h.Register(r)

	// 1. Login success
	bodyLogin, _ := json.Marshal(map[string]string{
		"username_or_email": "admin",
		"password":          "password123",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyLogin))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", wLogin.Code)
	}

	// 2. Login failure -> 401
	bodyBad, _ := json.Marshal(map[string]string{
		"username_or_email": "admin",
		"password":          "wrong",
	})
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bodyBad))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on bad login, got %d", wBad.Code)
	}

	// 3. Refresh success
	bodyRefresh, _ := json.Marshal(map[string]string{
		"refresh_token": "mock-refresh-token",
	})
	reqRefresh := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(bodyRefresh))
	reqRefresh.Header.Set("Content-Type", "application/json")
	wRefresh := httptest.NewRecorder()
	r.ServeHTTP(wRefresh, reqRefresh)

	if wRefresh.Code != http.StatusOK {
		t.Fatalf("expected 200 on refresh, got %d", wRefresh.Code)
	}

	// 4. Logout success
	bodyLogout, _ := json.Marshal(map[string]string{
		"session_id": "mock-session-id",
	})
	reqLogout := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", bytes.NewReader(bodyLogout))
	reqLogout.Header.Set("Content-Type", "application/json")
	wLogout := httptest.NewRecorder()
	r.ServeHTTP(wLogout, reqLogout)

	if wLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d", wLogout.Code)
	}
}
