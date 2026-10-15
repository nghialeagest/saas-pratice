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
	"saasbase/platform/errors"
)

type RoleService struct {
	api.UnimplementedRoleServiceServer
	repo *data.RoleRepo
}

func NewRoleService(repo *data.RoleRepo) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) ListRoles(ctx context.Context, _ *api.ListRolesRequest) (*api.ListRolesResponse, error) {
	roles, err := s.repo.List(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return &api.ListRolesResponse{
		Items: toRoleProtos(roles),
	}, nil
}

func (s *RoleService) CreateRole(ctx context.Context, req *api.CreateRoleRequest) (*api.Role, error) {
	if req.GetName() == "" {
		return nil, errors.BadRequest("role.invalid_argument", "role name is required")
	}

	role := &biz.Role{
		ID:             uuid.NewString(),
		Name:           req.GetName(),
		NormalizedName: strings.ToUpper(req.GetName()),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, role); err != nil {
		return nil, errors.Internal(err)
	}

	return toRoleProto(role), nil
}

func (s *RoleService) DeleteRole(ctx context.Context, req *api.DeleteRoleRequest) (*api.DeleteRoleResponse, error) {
	if req.GetId() == "" {
		return nil, errors.BadRequest("role.invalid_argument", "role id is required")
	}

	if _, err := s.repo.Get(ctx, req.GetId()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("role.not_found", "role not found")
		}
		return nil, errors.Internal(err)
	}

	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		return nil, errors.Internal(err)
	}

	return &api.DeleteRoleResponse{}, nil
}

func toRoleProto(r *biz.Role) *api.Role {
	if r == nil {
		return nil
	}
	return &api.Role{
		Id:   r.ID,
		Name: r.Name,
	}
}

func toRoleProtos(roles []*biz.Role) []*api.Role {
	out := make([]*api.Role, len(roles))
	for i, r := range roles {
		out[i] = toRoleProto(r)
	}
	return out
}
