package grpcx

import (
	"context"
	"errors"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	platformerrors "saasbase/platform/errors"
)

// ErrorUnaryInterceptor intercepts errors returned by gRPC handlers and converts them
// to appropriate gRPC status errors. For internal errors, error details are masked.
func ErrorUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		method := "unknown"
		if info != nil {
			method = info.FullMethod
		}

		var appErr *platformerrors.AppError
		if errors.As(err, &appErr) {
			grpcCode := appErr.GRPCCode()
			if grpcCode == codes.Internal || appErr.HTTPStatus >= 500 {
				log.Printf("[ERROR] grpc %s: %v", method, err)
				return nil, status.Error(codes.Internal, "internal server error")
			}
			msg := appErr.Message
			if msg == "" {
				msg = appErr.Code
			}
			return nil, status.Error(grpcCode, msg)
		}

		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.Internal || s.Code() == codes.Unknown {
				log.Printf("[ERROR] grpc %s: %v", method, err)
				return nil, status.Error(s.Code(), "internal server error")
			}
			return nil, err
		}

		log.Printf("[ERROR] grpc %s: %v", method, err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
}
