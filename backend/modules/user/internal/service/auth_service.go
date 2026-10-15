package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/platform/authn"
	"saasbase/platform/errors"
)

const (
	errInvalidCredentials = "invalid username, email or password"
	maxFailedAttempts     = 5
	lockoutDuration       = 15 * time.Minute
)

type AuthService struct {
	api.UnimplementedAuthServiceServer
	userRepo       *data.UserRepo
	userTenantRepo *data.UserTenantRepo
	sessionRepo    *data.SessionRepo
	hasher         authn.Hasher
	tokenCfg       authn.TokenConfig
}

func NewAuthService(
	userRepo *data.UserRepo,
	userTenantRepo *data.UserTenantRepo,
	sessionRepo *data.SessionRepo,
	hasher authn.Hasher,
	tokenCfg authn.TokenConfig,
) *AuthService {
	if hasher == nil {
		hasher = authn.NewHasher(12)
	}
	return &AuthService{
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		sessionRepo:    sessionRepo,
		hasher:         hasher,
		tokenCfg:       tokenCfg,
	}
}

func (s *AuthService) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	if req.GetUsernameOrEmail() == "" || req.GetPassword() == "" {
		return nil, errors.Unauthorized(errInvalidCredentials)
	}

	u, err := s.userRepo.FindByUsernameOrEmail(ctx, req.GetUsernameOrEmail())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Perform dummy hash verify to mitigate timing enumeration attacks
			_ = s.hasher.Verify("dummy", "$2a$12$e8k8yO.u0kR3k2W1Q2h6g.5ZkO6GvHwI5aM1x7Z6jY3s5uL9w6M7q")
			return nil, errors.Unauthorized(errInvalidCredentials)
		}
		return nil, errors.Internal(err)
	}

	// Check lockout
	if u.LockoutUntil != nil && time.Now().Before(*u.LockoutUntil) {
		return nil, errors.Unauthorized("account is temporarily locked due to too many failed attempts")
	}

	// Check active status
	if u.Status != biz.UserStatusActive {
		return nil, errors.Unauthorized("account is inactive or disabled")
	}

	// Verify password
	if !s.hasher.Verify(req.GetPassword(), u.PasswordHash) {
		u.AccessFailedCount++
		if u.AccessFailedCount >= maxFailedAttempts {
			lockout := time.Now().Add(lockoutDuration)
			u.LockoutUntil = &lockout
		}
		u.UpdatedAt = time.Now()
		_ = s.userRepo.Update(ctx, u)
		return nil, errors.Unauthorized(errInvalidCredentials)
	}

	// Reset failed attempts on success
	if u.AccessFailedCount > 0 || u.LockoutUntil != nil {
		u.AccessFailedCount = 0
		u.LockoutUntil = nil
		u.UpdatedAt = time.Now()
		_ = s.userRepo.Update(ctx, u)
	}

	// Verify tenant membership if a tenant is requested
	tenantID := req.GetTenantId()
	if tenantID != "" {
		ut, err := s.userTenantRepo.Get(ctx, u.ID, tenantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.Forbidden("auth.not_tenant_member", "user is not a member of the requested tenant")
			}
			return nil, errors.Internal(err)
		}
		if ut.Status != biz.MemberStatusActive {
			return nil, errors.Forbidden("auth.tenant_membership_inactive", "membership in this tenant is not active")
		}
	}

	// Create session & tokens
	sessionID := uuid.NewString()
	refreshToken := uuid.NewString() + "-" + uuid.NewString()
	session := &biz.Session{
		ID:           sessionID,
		UserID:       u.ID,
		TenantID:     tenantID,
		Username:     u.Username,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, errors.Internal(err)
	}

	accessToken, err := authn.IssueAccess(s.tokenCfg, u.ID, tenantID, sessionID, u.Username)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return &api.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.tokenCfg.AccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *api.RefreshRequest) (*api.RefreshResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, errors.Unauthorized("refresh token is required")
	}

	session, err := s.sessionRepo.FindByRefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Unauthorized("invalid or revoked refresh token")
		}
		return nil, errors.Internal(err)
	}

	if !session.IsActive() {
		return nil, errors.Unauthorized("session is expired or has been revoked")
	}

	// Verify user is still active
	u, err := s.userRepo.Get(ctx, session.UserID)
	if err != nil || u.Status != biz.UserStatusActive {
		return nil, errors.Unauthorized("user is inactive")
	}

	// Verify tenant membership is still active
	if session.TenantID != "" {
		ut, err := s.userTenantRepo.Get(ctx, session.UserID, session.TenantID)
		if err != nil || ut.Status != biz.MemberStatusActive {
			return nil, errors.Forbidden("auth.tenant_membership_inactive", "tenant membership is no longer active")
		}
	}

	// Token rotation: generate a NEW refresh token and update the session
	newRefreshToken := uuid.NewString() + "-" + uuid.NewString()
	session.RefreshToken = newRefreshToken
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, errors.Internal(err)
	}

	newAccessToken, err := authn.IssueAccess(s.tokenCfg, session.UserID, session.TenantID, session.ID, u.Username)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return &api.RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.tokenCfg.AccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	if req.GetSessionId() != "" {
		_ = s.sessionRepo.Revoke(ctx, req.GetSessionId())
	}
	return &api.LogoutResponse{}, nil
}
