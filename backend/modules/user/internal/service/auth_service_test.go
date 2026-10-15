package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/modules/user/internal/service"
	"saasbase/platform/authn"
	"saasbase/platform/errors"
)

type authTestEnv struct {
	authSvc        *service.AuthService
	userRepo       *data.UserRepo
	userTenantRepo *data.UserTenantRepo
	sessionRepo    *data.SessionRepo
	hasher         authn.Hasher
	tokenCfg       authn.TokenConfig
}

func setupAuthEnv(t *testing.T) *authTestEnv {
	t.Helper()
	provider := setupTestDB(t)

	hasher := authn.NewHasher(12)
	tokenCfg := authn.DefaultTokenConfig()

	userRepo := data.NewUserRepo(provider)
	userTenantRepo := data.NewUserTenantRepo(provider)
	sessionRepo := data.NewSessionRepo(provider)

	authSvc := service.NewAuthService(userRepo, userTenantRepo, sessionRepo, hasher, tokenCfg)

	return &authTestEnv{
		authSvc:        authSvc,
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		sessionRepo:    sessionRepo,
		hasher:         hasher,
		tokenCfg:       tokenCfg,
	}
}

func TestAuthService_Login_SuccessAndFailure(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	// Create test user
	pwdHash, _ := env.hasher.Hash("CorrectPassword123!")
	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           "johndoe",
		NormalizedUsername: "JOHNDOE",
		Email:              "john@example.com",
		NormalizedEmail:    "JOHN@EXAMPLE.COM",
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := env.userRepo.Create(ctx, u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Join tenant-alpha
	ut := &biz.UserTenant{
		UserID:    u.ID,
		TenantID:  "tenant-alpha",
		Status:    biz.MemberStatusActive,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := env.userTenantRepo.Create(ctx, ut); err != nil {
		t.Fatalf("failed to join tenant: %v", err)
	}

	// 1. Correct login with username
	resp, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "johndoe",
		Password:        "CorrectPassword123!",
		TenantId:        "tenant-alpha",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got access=%q, refresh=%q", resp.AccessToken, resp.RefreshToken)
	}

	// 2. Correct login with email
	respEmail, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "john@example.com",
		Password:        "CorrectPassword123!",
		TenantId:        "tenant-alpha",
	})
	if err != nil {
		t.Fatalf("login with email failed: %v", err)
	}
	if respEmail.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}

	// 3. Wrong password -> 401 with generic error message
	_, errWrongPwd := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "johndoe",
		Password:        "WrongPassword!",
		TenantId:        "tenant-alpha",
	})
	if errWrongPwd == nil {
		t.Fatal("expected wrong password to fail, but succeeded")
	}

	// 4. Non-existent user -> 401 with IDENTICAL error message (no username enumeration)
	_, errNonExistent := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "ghost_user",
		Password:        "WrongPassword!",
		TenantId:        "tenant-alpha",
	})
	if errNonExistent == nil {
		t.Fatal("expected non-existent user to fail, but succeeded")
	}

	if errWrongPwd.Error() != errNonExistent.Error() {
		t.Fatalf("SECURITY VIOLATION: error messages differ between wrong password and wrong username: %q vs %q",
			errWrongPwd.Error(), errNonExistent.Error())
	}
}

func TestAuthService_Login_TenantMembershipRequired(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	pwdHash, _ := env.hasher.Hash("CorrectPassword123!")
	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           "tenantuser",
		NormalizedUsername: "TENANTUSER",
		Email:              "tenantuser@example.com",
		NormalizedEmail:    "TENANTUSER@EXAMPLE.COM",
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_ = env.userRepo.Create(ctx, u)

	// User belongs only to tenant-first
	_ = env.userTenantRepo.Create(ctx, &biz.UserTenant{
		UserID:    u.ID,
		TenantID:  "tenant-first",
		Status:    biz.MemberStatusActive,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Try logging into tenant-second (not a member) -> MUST BE REJECTED
	_, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "tenantuser",
		Password:        "CorrectPassword123!",
		TenantId:        "tenant-second",
	})
	if err == nil {
		t.Fatal("expected login to tenant-second to fail, but succeeded")
	}

	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		if appErr.HTTPStatus != 403 {
			t.Fatalf("expected HTTP 403 Forbidden for non-member tenant login, got %d", appErr.HTTPStatus)
		}
	}
}

func TestAuthService_PasswordLockout(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	pwdHash, _ := env.hasher.Hash("CorrectPassword123!")
	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           "lockoutuser",
		NormalizedUsername: "LOCKOUTUSER",
		Email:              "lockout@example.com",
		NormalizedEmail:    "LOCKOUT@EXAMPLE.COM",
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_ = env.userRepo.Create(ctx, u)

	// Fail password 5 times
	for i := 0; i < 5; i++ {
		_, _ = env.authSvc.Login(ctx, &api.LoginRequest{
			UsernameOrEmail: "lockoutuser",
			Password:        "BadPassword",
		})
	}

	// 6th attempt with CORRECT password must be rejected because account is locked
	_, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "lockoutuser",
		Password:        "CorrectPassword123!",
	})
	if err == nil {
		t.Fatal("expected locked account to reject login even with correct password")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "locked") {
		t.Fatalf("expected lockout message, got: %v", err)
	}
}

func TestAuthService_RefreshToken_RotationAndReuseRejected(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	pwdHash, _ := env.hasher.Hash("CorrectPassword123!")
	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           "rotationuser",
		NormalizedUsername: "ROTATIONUSER",
		Email:              "rotation@example.com",
		NormalizedEmail:    "ROTATION@EXAMPLE.COM",
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_ = env.userRepo.Create(ctx, u)

	loginResp, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "rotationuser",
		Password:        "CorrectPassword123!",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	refreshToken1 := loginResp.RefreshToken

	// 1. First refresh with refreshToken1 -> MUST SUCCEED and return refreshToken2
	refResp1, err := env.authSvc.Refresh(ctx, &api.RefreshRequest{
		RefreshToken: refreshToken1,
	})
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
	refreshToken2 := refResp1.RefreshToken

	if refreshToken1 == refreshToken2 {
		t.Fatalf("expected rotated refresh token to be different from original token")
	}

	// 2. REUSE: Calling refresh with refreshToken1 a second time -> MUST BE REJECTED!
	_, errReuse := env.authSvc.Refresh(ctx, &api.RefreshRequest{
		RefreshToken: refreshToken1,
	})
	if errReuse == nil {
		t.Fatal("SECURITY VIOLATION: Reused old refresh token succeeded, rotation did not invalidate old token!")
	}

	// 3. Refresh with the new refreshToken2 -> MUST SUCCEED
	refResp2, err := env.authSvc.Refresh(ctx, &api.RefreshRequest{
		RefreshToken: refreshToken2,
	})
	if err != nil {
		t.Fatalf("refresh with rotated token failed: %v", err)
	}
	if refResp2.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}

func TestAuthService_Logout_RevokesSession(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	pwdHash, _ := env.hasher.Hash("CorrectPassword123!")
	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           "logoutuser",
		NormalizedUsername: "LOGOUTUSER",
		Email:              "logout@example.com",
		NormalizedEmail:    "LOGOUT@EXAMPLE.COM",
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_ = env.userRepo.Create(ctx, u)

	loginResp, err := env.authSvc.Login(ctx, &api.LoginRequest{
		UsernameOrEmail: "logoutuser",
		Password:        "CorrectPassword123!",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	claims, err := authn.ParseAccess(env.tokenCfg, loginResp.AccessToken)
	if err != nil {
		t.Fatalf("failed to parse access token: %v", err)
	}

	// Logout the session
	_, err = env.authSvc.Logout(ctx, &api.LogoutRequest{
		SessionId: claims.SessionID,
	})
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Refresh attempt after logout must fail
	_, err = env.authSvc.Refresh(ctx, &api.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	if err == nil {
		t.Fatal("expected refresh to fail after logout, but succeeded")
	}
}
