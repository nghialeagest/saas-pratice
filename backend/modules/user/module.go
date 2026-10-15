package user

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"saasbase/modules/user/api"
	"saasbase/modules/user/internal/biz"
	"saasbase/modules/user/internal/data"
	"saasbase/modules/user/internal/handler"
	"saasbase/modules/user/internal/service"
	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/grpcx"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
)

type Module struct {
	userSvc     api.UserServiceServer
	roleSvc     api.RoleServiceServer
	internalSvc api.UserInternalServiceServer
	authSvc     api.AuthServiceServer
	sessionRepo *data.SessionRepo
	permRepo    *data.PermissionRepo
	authHandler *handler.AuthHandler
	userHandler *handler.UserHandler
	roleHandler *handler.RoleHandler
	checker     *authz.CachedChecker
	eventBus    authz.EventBus
}

var (
	_ module.Module   = (*Module)(nil)
	_ module.Seeder   = (*Module)(nil)
	_ httpx.Registrar = (*Module)(nil)
	_ grpcx.Registrar = (*Module)(nil)
)

func init() { module.Register(&Module{}) }

func (Module) Name() string { return "user" }

func (Module) Tables() []any {
	return []any{
		&biz.User{},
		&biz.Role{},
		&biz.UserRole{},
		&biz.UserTenant{},
		&biz.Session{},
		&biz.Permission{},
		&biz.RolePermission{},
	}
}

func (Module) Permissions() []module.Permission { return api.Permissions() }

func (m *Module) Init(deps module.Deps) error {
	hasher := authn.NewHasher(12)
	tokenCfg := authn.DefaultTokenConfig()

	m.eventBus = authz.NewMemoryEventBus()
	m.permRepo = data.NewPermissionRepo(deps.DB, m.eventBus)

	userRepo := data.NewUserRepo(deps.DB)
	roleRepo := data.NewRoleRepo(deps.DB)
	userRoleRepo := data.NewUserRoleRepo(deps.DB, m.eventBus)
	userTenantRepo := data.NewUserTenantRepo(deps.DB)
	m.sessionRepo = data.NewSessionRepo(deps.DB)

	m.userSvc = service.NewUserService(userRepo, userRoleRepo, roleRepo, userTenantRepo, hasher)
	m.roleSvc = service.NewRoleService(roleRepo)
	m.internalSvc = service.NewUserInternalService(userRepo, userTenantRepo, roleRepo, userRoleRepo, m.permRepo, m.eventBus, hasher)
	m.authSvc = service.NewAuthService(userRepo, userTenantRepo, m.sessionRepo, hasher, tokenCfg)
	m.authHandler = handler.NewAuthHandler(m.authSvc)

	m.checker = authz.NewCachedChecker(m.permRepo, m.eventBus, 5*time.Minute)
	authz.SetDefaultChecker(m.checker)

	m.userHandler = handler.NewUserHandler(m.userSvc)
	m.roleHandler = handler.NewRoleHandler(m.roleSvc, m.permRepo)

	module.Provide(m.userSvc)
	module.Provide(m.roleSvc)
	module.Provide(m.internalSvc)
	module.Provide(m.authSvc)
	module.Provide(authn.SessionValidator(m))
	module.Provide(authz.Checker(m.checker))
	module.Provide(m.permRepo)
	module.Provide(api.PermissionManager(m.permRepo))
	module.Provide(m.eventBus)

	return nil
}

// Seed synchronizes module permissions into the tenant-scoped permissions table.
func (m *Module) Seed(ctx context.Context, tenantID string) error {
	if m.permRepo == nil {
		return nil
	}
	return m.permRepo.SyncPermissions(ctx, tenantID, module.Permissions())
}

func (m *Module) RegisterHTTP(r gin.IRouter) {
	if m.authHandler != nil {
		m.authHandler.Register(r)
	}
	if m.userHandler != nil {
		m.userHandler.Register(r)
	}
	if m.roleHandler != nil {
		m.roleHandler.Register(r)
	}
}

func (m *Module) RegisterGRPC(s grpc.ServiceRegistrar) {
	api.RegisterUserServiceServer(s, m.userSvc)
	api.RegisterRoleServiceServer(s, m.roleSvc)
	api.RegisterUserInternalServiceServer(s, m.internalSvc)
	api.RegisterAuthServiceServer(s, m.authSvc)
}

func (m *Module) UserService() api.UserServiceServer             { return m.userSvc }
func (m *Module) RoleService() api.RoleServiceServer             { return m.roleSvc }
func (m *Module) InternalService() api.UserInternalServiceServer { return m.internalSvc }
func (m *Module) AuthService() api.AuthServiceServer             { return m.authSvc }
func (m *Module) Checker() authz.Checker                         { return m.checker }
func (m *Module) PermissionRepo() *data.PermissionRepo           { return m.permRepo }

// ValidateSession implements authn.SessionValidator for session revocation checks.
func (m *Module) ValidateSession(ctx context.Context, sessionID string) bool {
	if m.sessionRepo == nil {
		return false
	}
	s, err := m.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return false
	}
	return s.IsActive()
}
