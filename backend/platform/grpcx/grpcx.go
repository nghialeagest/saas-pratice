package grpcx

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"saasbase/platform/tenant"
)

// Registrar is implemented by modules that expose an internal gRPC surface.
type Registrar interface {
	RegisterGRPC(s grpc.ServiceRegistrar)
}

// TenantUnaryInterceptor extracts x-tenant-id from gRPC metadata.
// It is fail-closed: if the tenant metadata is missing and the method is not exempt,
// it rejects the request with codes.InvalidArgument rather than falling back to host scope.
func TenantUnaryInterceptor(exemptMethods ...string) grpc.UnaryServerInterceptor {
	exemptMap := make(map[string]bool, len(exemptMethods))
	for _, m := range exemptMethods {
		exemptMap[m] = true
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info != nil && exemptMap[info.FullMethod] {
			return handler(ctx, req)
		}

		var tid string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get(tenant.MetadataKey()); len(v) > 0 {
				tid = v[0]
			}
		}

		if tid == "" {
			return nil, status.Error(codes.InvalidArgument, "missing x-tenant-id metadata")
		}

		ctx = tenant.NewContext(ctx, tid)
		return handler(ctx, req)
	}
}

// TenantClientUnaryInterceptor automatically propagates the tenant ID from context
// into outgoing gRPC metadata across module/network boundaries.
func TenantClientUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if tid := tenant.FromContext(ctx); tid != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, tenant.MetadataKey(), tid)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
