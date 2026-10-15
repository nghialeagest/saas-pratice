package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	userapi "saasbase/modules/user/api"
	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/data"
	"saasbase/platform/grpcx"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
	"saasbase/platform/observability"
	"saasbase/platform/tenant"

	_ "saasbase/modules/note"
	_ "saasbase/modules/task"
	_ "saasbase/modules/tenant"
	_ "saasbase/modules/user"
)

var publicPrefixes = []string{"/healthz", "/v1/auth"}

func defaultPorts(mod string) (string, string) {
	switch strings.ToLower(mod) {
	case "user":
		return ":8081", ":9091"
	case "note":
		return ":8082", ":9092"
	case "task":
		return ":8083", ":9093"
	case "tenant":
		return ":8084", ":9094"
	default:
		return ":8080", ":9090"
	}
}

func main() {
	moduleFlag := flag.String("module", "", "Target module to run (user, note, task, tenant, all)")
	flag.Parse()

	targetModule := *moduleFlag
	if targetModule == "" {
		targetModule = env("MODULE", "all")
	}
	targetModule = strings.ToLower(targetModule)

	serviceName := "saasbase"
	if targetModule != "" && targetModule != "all" {
		serviceName = "saasbase-" + targetModule
	}

	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	slog.SetDefault(logger)

	_, shutdownTracer, err := observability.InitTracer(serviceName, nil)
	if err != nil {
		logger.Error("failed to initialize tracer", "error", err)
	}

	defaultHTTP, defaultGRPC := defaultPorts(targetModule)
	dsn := env("DATABASE_DSN", "postgres://postgres:postgres@localhost:55432/saas?sslmode=disable")
	httpAddr := env("HTTP_ADDR", defaultHTTP)
	grpcAddr := env("GRPC_ADDR", defaultGRPC)

	provider := data.NewProvider(data.NewSharedConn(dsn), nil)
	defer provider.Close()

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		logger.Error("failed to initialize modules", "error", err)
		os.Exit(1)
	}

	bootCtx := tenant.SkipFilter(tenant.NewContext(context.Background(), ""))
	db, err := provider.DB(bootCtx)
	if err != nil {
		logger.Error("failed to get database handle", "error", err)
		os.Exit(1)
	}
	_ = observability.RegisterGORMTracing(db)

	if err := db.AutoMigrate(module.Tables()...); err != nil {
		logger.Error("failed to auto migrate tables", "error", err)
		os.Exit(1)
	}

	// Enable PostgreSQL Row Level Security on tenant-scoped tables
	if err := tenant.EnableRLS(db, module.Tables()...); err != nil {
		logger.Warn("enable rls warning", "error", err)
	}

	if targetModule == "all" || targetModule == "user" {
		seedDefaultData(bootCtx, db, logger)
	}

	authz.MustHaveChecker()

	httpSrv := &http.Server{Addr: httpAddr, Handler: newRouter(targetModule)}
	grpcSrv := newGRPCServer(targetModule)

	go func() {
		logger.Info("http server listening", "service", serviceName, "addr", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Error("failed to listen tcp", "error", err)
			os.Exit(1)
		}
		logger.Info("grpc server listening", "service", serviceName, "addr", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("grpc server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if shutdownTracer != nil {
		_ = shutdownTracer(ctx)
	}
	_ = httpSrv.Shutdown(ctx)
	grpcSrv.GracefulStop()
}

func newRouter(targetModule ...string) *gin.Engine {
	mod := "all"
	if len(targetModule) > 0 && targetModule[0] != "" {
		mod = targetModule[0]
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	var sessionValidator authn.SessionValidator
	if sv, err := module.Resolve[authn.SessionValidator](); err == nil {
		sessionValidator = sv
	}

	tokenCfg := authn.DefaultTokenConfig()
	authnMid := authn.GinMiddleware(authn.MiddlewareOptions{
		TokenConfig:      tokenCfg,
		SessionValidator: sessionValidator,
		PublicPrefixes:   publicPrefixes,
	})

	var tenantValidator tenant.Validator
	if internalSvc, err := module.Resolve[userapi.UserInternalServiceServer](); err == nil {
		tv := userapi.NewTenantValidator(internalSvc)
		if bus, err := module.Resolve[authz.EventBus](); err == nil {
			tv.SubscribeToEventBus(bus)
		}
		tenantValidator = tv
	}

	tenantMid := tenant.Middleware(tenant.Options{
		Validator:      tenantValidator,
		AllowHost:      true,
		PublicPrefixes: publicPrefixes,
	})

	// Pipeline order: recover -> cors -> tracing -> requestID -> authn -> tenant -> logger
	r.Use(
		gin.Recovery(),
		httpx.CORSMiddleware(httpx.DefaultCORSOptions()),
		observability.HTTPTraceMiddleware(),
		observability.RequestIDMiddleware(),
		authnMid,
		tenantMid,
		observability.HTTPLoggingMiddleware(slog.Default()),
	)

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	api := r.Group("/")
	for _, m := range module.All() {
		if mod != "" && mod != "all" && strings.ToLower(m.Name()) != mod {
			continue
		}
		if reg, ok := m.(httpx.Registrar); ok {
			reg.RegisterHTTP(api)
		}
	}
	return r
}

func newGRPCServer(targetModule ...string) *grpc.Server {
	mod := "all"
	if len(targetModule) > 0 && targetModule[0] != "" {
		mod = targetModule[0]
	}

	var sessionValidator authn.SessionValidator
	if sv, err := module.Resolve[authn.SessionValidator](); err == nil {
		sessionValidator = sv
	}
	tokenCfg := authn.DefaultTokenConfig()

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		observability.GRPCTraceInterceptor(),
		grpcx.ErrorUnaryInterceptor(),
		authn.GRPCUnaryInterceptor(tokenCfg, sessionValidator, "/user.v1.AuthService/Login", "/user.v1.AuthService/Refresh"),
		grpcx.TenantUnaryInterceptor("/user.v1.AuthService/Login", "/user.v1.AuthService/Refresh"),
	))
	for _, m := range module.All() {
		if mod != "" && mod != "all" && strings.ToLower(m.Name()) != mod {
			continue
		}
		if reg, ok := m.(grpcx.Registrar); ok {
			reg.RegisterGRPC(s)
		}
	}
	reflection.Register(s)
	return s
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
