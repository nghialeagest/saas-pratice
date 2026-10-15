package note

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"saasbase/modules/note/api"
	"saasbase/modules/note/internal/biz"
	"saasbase/modules/note/internal/data"
	"saasbase/modules/note/internal/handler"
	"saasbase/modules/note/internal/service"
	"saasbase/platform/grpcx"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
)

type Module struct {
	svc     api.NoteServiceServer
	handler *handler.Handler
}

var (
	_ module.Module    = (*Module)(nil)
	_ httpx.Registrar  = (*Module)(nil)
	_ grpcx.Registrar  = (*Module)(nil)
)

func init() { module.Register(&Module{}) }

func (Module) Name() string { return "note" }

func (Module) Tables() []any { return []any{&biz.Note{}} }

func (Module) Permissions() []module.Permission { return api.Permissions() }

func (m *Module) Init(deps module.Deps) error {
	m.svc = service.New(data.NewRepo(deps.DB))
	m.handler = handler.New(m.svc)
	module.Provide(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(r gin.IRouter) { m.handler.Register(r) }

func (m *Module) RegisterGRPC(s grpc.ServiceRegistrar) {
	api.RegisterNoteServiceServer(s, m.svc)
}

// Service exposes the module contract to other modules in the same process.
func (m *Module) Service() api.NoteServiceServer { return m.svc }
