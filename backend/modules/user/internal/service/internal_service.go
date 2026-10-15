package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/errors"
	"saasbase/platform/tenant"
)

type UserInternalService struct {
	api.UnimplementedUserInternalServiceServer
	userRepo       *data.UserRepo
	userTenantRepo *data.UserTenantRepo
	roleRepo       *data.RoleRepo
	userRoleRepo   *data.UserRoleRepo
	permRepo       *data.PermissionRepo
	eventBus       authz.EventBus
	hasher         authn.Hasher
}

func NewUserInternalService(
	userRepo *data.UserRepo,
	userTenantRepo *data.UserTenantRepo,
	roleRepo *data.RoleRepo,
	userRoleRepo *data.UserRoleRepo,
	permRepo *data.PermissionRepo,
	eventBus authz.EventBus,
	hasher authn.Hasher,
) *UserInternalService {
	if hasher == nil {
		hasher = authn.NewHasher(12)
	}
	return &UserInternalService{
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		roleRepo:       roleRepo,
		userRoleRepo:   userRoleRepo,
		permRepo:       permRepo,
		eventBus:       eventBus,
		hasher:         hasher,
	}
}

func (s *UserInternalService) CheckUserTenant(ctx context.Context, req *api.CheckUserTenantRequest) (*api.CheckUserTenantResponse, error) {
	if req.GetUserId() == "" || req.GetTenantId() == "" {
		return &api.CheckUserTenantResponse{Allowed: false}, nil
	}

	ut, err := s.userTenantRepo.Get(ctx, req.GetUserId(), req.GetTenantId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &api.CheckUserTenantResponse{Allowed: false}, nil
		}
		return nil, errors.Internal(err)
	}

	if ut.Status != biz.MemberStatusActive {
		return &api.CheckUserTenantResponse{Allowed: false}, nil
	}

	// Verify user is also active
	u, err := s.userRepo.Get(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &api.CheckUserTenantResponse{Allowed: false}, nil
		}
		return nil, errors.Internal(err)
	}

	if u.Status != biz.UserStatusActive {
		return &api.CheckUserTenantResponse{Allowed: false}, nil
	}

	return &api.CheckUserTenantResponse{Allowed: true}, nil
}

func (s *UserInternalService) ListUserTenants(ctx context.Context, req *api.ListUserTenantsRequest) (*api.ListUserTenantsResponse, error) {
	if req.GetUserId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "user_id is required")
	}

	list, err := s.userTenantRepo.ListByUser(ctx, req.GetUserId())
	if err != nil {
		return nil, errors.Internal(err)
	}

	items := make([]*api.TenantMembership, len(list))
	for i, ut := range list {
		status := api.MembershipStatus_MEMBERSHIP_STATUS_ACTIVE
		if ut.Status != biz.MemberStatusActive {
			status = api.MembershipStatus_MEMBERSHIP_STATUS_INACTIVE
		}
		items[i] = &api.TenantMembership{
			TenantId: ut.TenantID,
			Status:   status,
		}
	}

	return &api.ListUserTenantsResponse{Items: items}, nil
}

func (s *UserInternalService) JoinTenant(ctx context.Context, req *api.JoinTenantRequest) (*api.JoinTenantResponse, error) {
	if req.GetUserId() == "" || req.GetTenantId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "user_id and tenant_id are required")
	}

	existing, err := s.userTenantRepo.Get(ctx, req.GetUserId(), req.GetTenantId())
	if err == nil {
		if existing.Status != biz.MemberStatusActive {
			existing.Status = biz.MemberStatusActive
			existing.UpdatedAt = time.Now()
			if err := s.userTenantRepo.Update(ctx, existing); err != nil {
				return nil, errors.Internal(err)
			}
		}
		return &api.JoinTenantResponse{}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Internal(err)
	}

	ut := &biz.UserTenant{
		UserID:    req.GetUserId(),
		TenantID:  req.GetTenantId(),
		Status:    biz.MemberStatusActive,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userTenantRepo.Create(ctx, ut); err != nil {
		return nil, errors.Internal(err)
	}

	return &api.JoinTenantResponse{}, nil
}

// RemoveUserTenant removes a user's membership from a tenant and emits an invalidation event.
func (s *UserInternalService) RemoveUserTenant(ctx context.Context, userID, tenantID string) error {
	if userID == "" || tenantID == "" {
		return errors.BadRequest("user.invalid_argument", "user_id and tenant_id are required")
	}
	if err := s.userTenantRepo.Delete(ctx, userID, tenantID); err != nil {
		return errors.Internal(err)
	}
	if s.eventBus != nil {
		s.eventBus.Publish(ctx, authz.InvalidationEvent{
			Type:     authz.EventUserLeftTenant,
			UserID:   userID,
			TenantID: tenantID,
		})
	}
	return nil
}

func (s *UserInternalService) ProvisionTenantAdmin(ctx context.Context, req *api.ProvisionTenantAdminRequest) (*api.User, error) {
	if req.GetTenantId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "tenant_id is required")
	}
	if req.GetEmail() == "" || req.GetUsername() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "email and username are required")
	}

	normEmail := strings.ToUpper(req.GetEmail())
	u, err := s.userRepo.FindByNormalizedEmail(ctx, normEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pwdHash, err := s.hasher.Hash(req.GetPassword())
			if err != nil {
				return nil, errors.Internal(err)
			}
			u = &biz.User{
				ID:                 uuid.NewString(),
				Username:           req.GetUsername(),
				NormalizedUsername: strings.ToUpper(req.GetUsername()),
				Email:              req.GetEmail(),
				NormalizedEmail:    normEmail,
				PasswordHash:       pwdHash,
				Status:             biz.UserStatusActive,
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			if err := s.userRepo.Create(ctx, u); err != nil {
				return nil, errors.Internal(err)
			}
		} else {
			return nil, errors.Internal(err)
		}
	}

	// Join the tenant
	ut := &biz.UserTenant{
		UserID:    u.ID,
		TenantID:  req.GetTenantId(),
		Status:    biz.MemberStatusActive,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.userTenantRepo.Create(ctx, ut); err != nil && !strings.Contains(err.Error(), "duplicate") {
		// Ignore if already joined
	}

	// Switch context to target tenant to create/assign Admin role
	tenantCtx := tenant.NewContext(ctx, req.GetTenantId())
	adminRole, err := s.roleRepo.FindByName(tenantCtx, "Admin")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			adminRole = &biz.Role{
				ID:             uuid.NewString(),
				Name:           "Admin",
				NormalizedName: "ADMIN",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := s.roleRepo.Create(tenantCtx, adminRole); err != nil {
				return nil, errors.Internal(err)
			}
		} else {
			return nil, errors.Internal(err)
		}
	}

	if err := s.userRoleRepo.Assign(tenantCtx, u.ID, []string{adminRole.ID}); err != nil {
		return nil, errors.Internal(err)
	}

	if s.permRepo != nil {
		_ = s.permRepo.AssignPermissions(tenantCtx, adminRole.ID, req.GetTenantId(), []authz.Permission{
			{Namespace: "*", Action: "*"},
		})
	}

	return toUserProto(u, []*api.Role{toRoleProto(adminRole)}), nil
}
