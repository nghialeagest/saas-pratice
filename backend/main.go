package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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

func main() {
	logger := observability.NewLogger(os.Stdout, slog.LevelInfo)
	slog.SetDefault(logger)

	_, shutdownTracer, err := observability.InitTracer("saasbase", nil)
	if err != nil {
		logger.Error("failed to initialize tracer", "error", err)
	}

	dsn := env("DATABASE_DSN", "postgres://postgres:postgres@localhost:55432/saas?sslmode=disable")
	httpAddr := env("HTTP_ADDR", ":8080")
	grpcAddr := env("GRPC_ADDR", ":9090")

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

	seedDefaultData(bootCtx, db, logger)

	authz.MustHaveChecker()

	httpSrv := &http.Server{Addr: httpAddr, Handler: newRouter()}
	grpcSrv := newGRPCServer()

	go func() {
		logger.Info("http server listening", "addr", httpAddr)
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
		logger.Info("grpc server listening", "addr", grpcAddr)
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

func newRouter() *gin.Engine {
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
		if reg, ok := m.(httpx.Registrar); ok {
			reg.RegisterHTTP(api)
		}
	}
	return r
}

func newGRPCServer() *grpc.Server {
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

