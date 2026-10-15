package api

import (
	"context"

	"google.golang.org/grpc"
)

// --- UserService ClientProxy ---

type userClientProxy struct {
	cc UserServiceClient
}

var _ UserServiceServer = (*userClientProxy)(nil)

func NewUserClientProxy(cc UserServiceClient) UserServiceServer {
	return &userClientProxy{cc: cc}
}

func NewUserClientProxyFromConn(conn grpc.ClientConnInterface) UserServiceServer {
	return NewUserClientProxy(NewUserServiceClient(conn))
}

func (p *userClientProxy) ListUsers(ctx context.Context, in *ListUsersRequest) (*ListUsersResponse, error) {
	return p.cc.ListUsers(ctx, in)
}

func (p *userClientProxy) GetUser(ctx context.Context, in *GetUserRequest) (*User, error) {
	return p.cc.GetUser(ctx, in)
}

func (p *userClientProxy) CreateUser(ctx context.Context, in *CreateUserRequest) (*User, error) {
	return p.cc.CreateUser(ctx, in)
}

func (p *userClientProxy) UpdateUser(ctx context.Context, in *UpdateUserRequest) (*User, error) {
	return p.cc.UpdateUser(ctx, in)
}

func (p *userClientProxy) DeleteUser(ctx context.Context, in *DeleteUserRequest) (*DeleteUserResponse, error) {
	return p.cc.DeleteUser(ctx, in)
}

func (p *userClientProxy) AssignRoles(ctx context.Context, in *AssignRolesRequest) (*User, error) {
	return p.cc.AssignRoles(ctx, in)
}

// --- RoleService ClientProxy ---

type roleClientProxy struct {
	cc RoleServiceClient
}

var _ RoleServiceServer = (*roleClientProxy)(nil)

func NewRoleClientProxy(cc RoleServiceClient) RoleServiceServer {
	return &roleClientProxy{cc: cc}
}

func NewRoleClientProxyFromConn(conn grpc.ClientConnInterface) RoleServiceServer {
	return NewRoleClientProxy(NewRoleServiceClient(conn))
}

func (p *roleClientProxy) ListRoles(ctx context.Context, in *ListRolesRequest) (*ListRolesResponse, error) {
	return p.cc.ListRoles(ctx, in)
}

func (p *roleClientProxy) CreateRole(ctx context.Context, in *CreateRoleRequest) (*Role, error) {
	return p.cc.CreateRole(ctx, in)
}

func (p *roleClientProxy) DeleteRole(ctx context.Context, in *DeleteRoleRequest) (*DeleteRoleResponse, error) {
	return p.cc.DeleteRole(ctx, in)
}

// --- UserInternalService ClientProxy ---

type userInternalClientProxy struct {
	cc UserInternalServiceClient
}

var _ UserInternalServiceServer = (*userInternalClientProxy)(nil)

func NewUserInternalClientProxy(cc UserInternalServiceClient) UserInternalServiceServer {
	return &userInternalClientProxy{cc: cc}
}

func NewUserInternalClientProxyFromConn(conn grpc.ClientConnInterface) UserInternalServiceServer {
	return NewUserInternalClientProxy(NewUserInternalServiceClient(conn))
}

func (p *userInternalClientProxy) CheckUserTenant(ctx context.Context, in *CheckUserTenantRequest) (*CheckUserTenantResponse, error) {
	return p.cc.CheckUserTenant(ctx, in)
}

func (p *userInternalClientProxy) ListUserTenants(ctx context.Context, in *ListUserTenantsRequest) (*ListUserTenantsResponse, error) {
	return p.cc.ListUserTenants(ctx, in)
}

func (p *userInternalClientProxy) JoinTenant(ctx context.Context, in *JoinTenantRequest) (*JoinTenantResponse, error) {
	return p.cc.JoinTenant(ctx, in)
}

func (p *userInternalClientProxy) ProvisionTenantAdmin(ctx context.Context, in *ProvisionTenantAdminRequest) (*User, error) {
	return p.cc.ProvisionTenantAdmin(ctx, in)
}

// --- AuthService ClientProxy ---

type authClientProxy struct {
	cc AuthServiceClient
}

var _ AuthServiceServer = (*authClientProxy)(nil)

func NewAuthClientProxy(cc AuthServiceClient) AuthServiceServer {
	return &authClientProxy{cc: cc}
}

func NewAuthClientProxyFromConn(conn grpc.ClientConnInterface) AuthServiceServer {
	return NewAuthClientProxy(NewAuthServiceClient(conn))
}

func (p *authClientProxy) Login(ctx context.Context, in *LoginRequest) (*LoginResponse, error) {
	return p.cc.Login(ctx, in)
}

func (p *authClientProxy) Refresh(ctx context.Context, in *RefreshRequest) (*RefreshResponse, error) {
	return p.cc.Refresh(ctx, in)
}

func (p *authClientProxy) Logout(ctx context.Context, in *LogoutRequest) (*LogoutResponse, error) {
	return p.cc.Logout(ctx, in)
}
