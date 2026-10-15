package task

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	noteapi "saasbase/modules/note/api"
	"saasbase/modules/task/api"
	"saasbase/modules/task/internal/biz"
	"saasbase/modules/task/internal/data"
	"saasbase/modules/task/internal/handler"
	"saasbase/modules/task/internal/service"
	"saasbase/platform/grpcx"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
)

type Module struct {
	repo    biz.Repo
	svc     api.TaskServiceServer
	handler *handler.Handler
}

var (
	_ module.Module   = (*Module)(nil)
	_ module.Wirer    = (*Module)(nil)
	_ httpx.Registrar = (*Module)(nil)
	_ grpcx.Registrar = (*Module)(nil)
)

func init() { module.Register(&Module{}) }

func (Module) Name() string { return "task" }

func (Module) Tables() []any { return []any{&biz.Task{}} }

func (Module) Permissions() []module.Permission { return api.Permissions() }

func (m *Module) Init(deps module.Deps) error {
	m.repo = data.NewRepo(deps.DB)
	return nil
}

func (m *Module) Wire() error {
	notes, err := module.Resolve[noteapi.NoteServiceServer]()
	if err != nil {
		return err
	}
	m.svc = service.New(m.repo, notes)
	m.handler = handler.New(m.svc)
	module.Provide(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(r gin.IRouter) { m.handler.Register(r) }

func (m *Module) RegisterGRPC(s grpc.ServiceRegistrar) {
	api.RegisterTaskServiceServer(s, m.svc)
}

func (m *Module) Service() api.TaskServiceServer { return m.svc }
