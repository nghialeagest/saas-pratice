package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/platform/authn"
	"saasbase/platform/errors"
	"saasbase/platform/tenant"
)

type UserService struct {
	api.UnimplementedUserServiceServer
	userRepo       *data.UserRepo
	userRoleRepo   *data.UserRoleRepo
	roleRepo       *data.RoleRepo
	userTenantRepo *data.UserTenantRepo
	hasher         authn.Hasher
}

func NewUserService(
	userRepo *data.UserRepo,
	userRoleRepo *data.UserRoleRepo,
	roleRepo *data.RoleRepo,
	userTenantRepo *data.UserTenantRepo,
	hasher authn.Hasher,
) *UserService {
	if hasher == nil {
		hasher = authn.NewHasher(12)
	}
	return &UserService{
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		roleRepo:       roleRepo,
		userTenantRepo: userTenantRepo,
		hasher:         hasher,
	}
}

func (s *UserService) ListUsers(ctx context.Context, req *api.ListUsersRequest) (*api.ListUsersResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 20
	}

	users, total, err := s.userRepo.List(ctx, page, pageSize, req.GetSearch())
	if err != nil {
		return nil, errors.Internal(err)
	}

	items := make([]*api.User, len(users))
	for i, u := range users {
		roles, err := s.userRoleRepo.ListRolesByUser(ctx, u.ID)
		if err != nil {
			return nil, errors.Internal(err)
		}
		items[i] = toUserProto(u, toRoleProtos(roles))
	}

	return &api.ListUsersResponse{
		Items: items,
		Total: total,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *api.GetUserRequest) (*api.User, error) {
	if req.GetId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "id is required")
	}

	u, err := s.userRepo.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user.not_found", "user not found")
		}
		return nil, errors.Internal(err)
	}

	roles, err := s.userRoleRepo.ListRolesByUser(ctx, u.ID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return toUserProto(u, toRoleProtos(roles)), nil
}

func (s *UserService) CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.User, error) {
	if req.GetUsername() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "username is required")
	}
	if req.GetEmail() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "email is required")
	}

	pwdHash := ""
	if req.GetPassword() != "" {
		var err error
		pwdHash, err = s.hasher.Hash(req.GetPassword())
		if err != nil {
			return nil, errors.Internal(err)
		}
	}

	u := &biz.User{
		ID:                 uuid.NewString(),
		Username:           req.GetUsername(),
		NormalizedUsername: strings.ToUpper(req.GetUsername()),
		Email:              req.GetEmail(),
		NormalizedEmail:    strings.ToUpper(req.GetEmail()),
		Name:               req.GetName(),
		Gender:             protoToGender(req.GetGender()),
		PasswordHash:       pwdHash,
		Status:             biz.UserStatusActive,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		if errors.IsDuplicateKey(err) {
			return nil, errors.Conflict("user.already_exists", "username or email already in use")
		}
		return nil, errors.Internal(err)
	}

	// users is a global table; without a membership row the account exists but
	// cannot sign in to the tenant it was just created in.
	if tid := tenant.FromContext(ctx); tid != "" && s.userTenantRepo != nil {
		if err := s.userTenantRepo.Create(ctx, &biz.UserTenant{
			UserID:    u.ID,
			TenantID:  tid,
			Status:    biz.MemberStatusActive,
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}); err != nil && !errors.IsDuplicateKey(err) {
			return nil, errors.Internal(err)
		}
	}

	return toUserProto(u, nil), nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *api.UpdateUserRequest) (*api.User, error) {
	if req.GetId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "id is required")
	}

	u, err := s.userRepo.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user.not_found", "user not found")
		}
		return nil, errors.Internal(err)
	}

	u.Name = req.GetName()
	if req.GetGender() != api.Gender_GENDER_UNSPECIFIED {
		u.Gender = protoToGender(req.GetGender())
	}
	u.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, errors.Internal(err)
	}

	roles, err := s.userRoleRepo.ListRolesByUser(ctx, u.ID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return toUserProto(u, toRoleProtos(roles)), nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *api.DeleteUserRequest) (*api.DeleteUserResponse, error) {
	if req.GetId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "id is required")
	}
	if err := s.userRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, errors.Internal(err)
	}
	return &api.DeleteUserResponse{}, nil
}

func (s *UserService) AssignRoles(ctx context.Context, req *api.AssignRolesRequest) (*api.User, error) {
	if req.GetUserId() == "" {
		return nil, errors.BadRequest("user.invalid_argument", "user_id is required")
	}

	u, err := s.userRepo.Get(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user.not_found", "user not found")
		}
		return nil, errors.Internal(err)
	}

	if err := s.userRoleRepo.Assign(ctx, u.ID, req.GetRoleIds()); err != nil {
		return nil, errors.Internal(err)
	}

	roles, err := s.userRoleRepo.ListRolesByUser(ctx, u.ID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return toUserProto(u, toRoleProtos(roles)), nil
}

// toUserProto converts a biz.User model to protobuf api.User message.
// PasswordHash is intentionally excluded to guarantee password hashes are never leaked.
func toUserProto(u *biz.User, roles []*api.Role) *api.User {
	if u == nil {
		return nil
	}
	return &api.User{
		Id:        u.ID,
		Username:  u.Username,
		Name:      u.Name,
		Email:     u.Email,
		Gender:    genderToProto(u.Gender),
		Roles:     roles,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}

func genderToProto(g string) api.Gender {
	switch strings.ToUpper(g) {
	case "MALE":
		return api.Gender_GENDER_MALE
	case "FEMALE":
		return api.Gender_GENDER_FEMALE
	case "OTHER":
		return api.Gender_GENDER_OTHER
	default:
		return api.Gender_GENDER_UNSPECIFIED
	}
}

func protoToGender(g api.Gender) string {
	switch g {
	case api.Gender_GENDER_MALE:
		return "MALE"
	case api.Gender_GENDER_FEMALE:
		return "FEMALE"
	case api.Gender_GENDER_OTHER:
		return "OTHER"
	default:
		return ""
	}
}
