package observability

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"

	"saasbase/platform/tenant"
)

const TracerName = "saasbase/observability"

var tracer = otel.Tracer(TracerName)

// InitTracer initializes OpenTelemetry tracer provider and propagator.
func InitTracer(serviceName string, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if serviceName == "" {
		serviceName = "saasbase"
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create otel resource: %w", err)
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	if exporter != nil {
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, tp.Shutdown, nil
}

// HTTPTraceMiddleware traces HTTP requests using OpenTelemetry.
func HTTPTraceMiddleware() gin.HandlerFunc {
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLPath(c.Request.URL.Path),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		span.SetAttributes(
			semconv.HTTPResponseStatusCode(c.Writer.Status()),
		)
		if tid := tenant.FromContext(c.Request.Context()); tid != "" {
			span.SetAttributes(attribute.String("tenant.id", tid))
		}
	}
}

// GRPCTraceInterceptor traces gRPC unary requests.
func GRPCTraceInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))
		}

		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(info.FullMethod),
			),
		)
		defer span.End()

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
		}
		return resp, err
	}
}

type metadataCarrier metadata.MD

func (m metadataCarrier) Get(key string) string {
	vals := metadata.MD(m).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (m metadataCarrier) Set(key, val string) {
	metadata.MD(m).Set(key, val)
}

func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// RegisterGORMTracing hooks OpenTelemetry spans into GORM queries.
func RegisterGORMTracing(db *gorm.DB) error {
	beforeHook := func(op string) func(db *gorm.DB) {
		return func(db *gorm.DB) {
			if db.Statement.Context == nil {
				return
			}
			ctx, span := tracer.Start(db.Statement.Context, "gorm:"+op,
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(
					attribute.String("db.system", "postgresql"),
					attribute.String("db.operation", op),
					attribute.String("db.table", db.Statement.Table),
				),
			)
			db.Statement.Context = ctx
			db.InstanceSet("otel:span", span)
		}
	}

	afterHook := func(db *gorm.DB) {
		if val, ok := db.InstanceGet("otel:span"); ok {
			if span, ok := val.(trace.Span); ok {
				if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
					span.RecordError(db.Error)
				}
				span.End()
			}
		}
	}

	_ = db.Callback().Create().Before("gorm:create").Register("otel:before_create", beforeHook("create"))
	_ = db.Callback().Create().After("gorm:create").Register("otel:after_create", afterHook)

	_ = db.Callback().Query().Before("gorm:query").Register("otel:before_query", beforeHook("query"))
	_ = db.Callback().Query().After("gorm:query").Register("otel:after_query", afterHook)

	_ = db.Callback().Update().Before("gorm:update").Register("otel:before_update", beforeHook("update"))
	_ = db.Callback().Update().After("gorm:update").Register("otel:after_update", afterHook)

	_ = db.Callback().Delete().Before("gorm:delete").Register("otel:before_delete", beforeHook("delete"))
	_ = db.Callback().Delete().After("gorm:delete").Register("otel:after_delete", afterHook)

	return nil
}
