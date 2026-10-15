package tenant

import (
	"github.com/gin-gonic/gin"

	"saasbase/modules/tenant/api"
	"saasbase/modules/tenant/internal/biz"
	"saasbase/modules/tenant/internal/data"
	"saasbase/modules/tenant/internal/handler"
	"saasbase/modules/tenant/internal/service"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
)

type Module struct {
	svc     api.TenantServiceServer
	handler *handler.TenantHandler
}

var (
	_ module.Module   = (*Module)(nil)
	_ httpx.Registrar = (*Module)(nil)
)

func init() { module.Register(&Module{}) }

func (Module) Name() string { return "tenant" }

func (Module) Tables() []any { return []any{&biz.Tenant{}} }

func (Module) Permissions() []module.Permission { return api.Permissions() }

func (m *Module) Init(deps module.Deps) error {
	repo := data.NewTenantRepo(deps.DB)
	m.svc = service.NewTenantService(repo)
	m.handler = handler.NewTenantHandler(m.svc)

	module.Provide(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(r gin.IRouter) {
	if m.handler != nil {
		m.handler.Register(r)
	}
}

func (m *Module) Service() api.TenantServiceServer { return m.svc }
