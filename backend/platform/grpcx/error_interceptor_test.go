package grpcx_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	platformerrors "saasbase/platform/errors"
	"saasbase/platform/grpcx"
)

func TestErrorUnaryInterceptor_Success(t *testing.T) {
	interceptor := grpcx.ErrorUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	resp, err := interceptor(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected 'ok', got %v", resp)
	}
}

func TestErrorUnaryInterceptor_AppErrorNotFound(t *testing.T) {
	interceptor := grpcx.ErrorUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, platformerrors.NotFound("note.not_found", "note not found")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetNote"}
	_, err := interceptor(context.Background(), "req", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected code NotFound, got %v", st.Code())
	}
	if st.Message() != "note not found" {
		t.Errorf("expected message 'note not found', got %q", st.Message())
	}
}

func TestErrorUnaryInterceptor_AppErrorInternalMasked(t *testing.T) {
	interceptor := grpcx.ErrorUnaryInterceptor()
	origErr := errors.New("sensitive postgres password failure")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, platformerrors.Internal(origErr)
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetNote"}
	_, err := interceptor(context.Background(), "req", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected code Internal, got %v", st.Code())
	}
	if st.Message() != "internal server error" {
		t.Errorf("expected message 'internal server error', got %q", st.Message())
	}
}

func TestErrorUnaryInterceptor_RawErrorMasked(t *testing.T) {
	interceptor := grpcx.ErrorUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("raw unhandled database crash")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetNote"}
	_, err := interceptor(context.Background(), "req", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected code Internal, got %v", st.Code())
	}
	if st.Message() != "internal server error" {
		t.Errorf("expected message 'internal server error', got %q", st.Message())
	}
}

func TestErrorUnaryInterceptor_GRPCStatusPreserved(t *testing.T) {
	interceptor := grpcx.ErrorUnaryInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid argument provided")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	_, err := interceptor(context.Background(), "req", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected code InvalidArgument, got %v", st.Code())
	}
	if st.Message() != "invalid argument provided" {
		t.Errorf("expected message 'invalid argument provided', got %q", st.Message())
	}
}
