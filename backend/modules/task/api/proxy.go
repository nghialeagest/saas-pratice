package api

import (
	"context"

	"google.golang.org/grpc"
)

type clientProxy struct {
	cc TaskServiceClient
}

var _ TaskServiceServer = (*clientProxy)(nil)

func NewClientProxy(cc TaskServiceClient) TaskServiceServer {
	return &clientProxy{cc: cc}
}

func NewClientProxyFromConn(conn grpc.ClientConnInterface) TaskServiceServer {
	return NewClientProxy(NewTaskServiceClient(conn))
}

func (p *clientProxy) CreateTask(ctx context.Context, in *CreateTaskRequest) (*Task, error) {
	return p.cc.CreateTask(ctx, in)
}

func (p *clientProxy) GetTask(ctx context.Context, in *GetTaskRequest) (*Task, error) {
	return p.cc.GetTask(ctx, in)
}

func (p *clientProxy) CountTasks(ctx context.Context, in *CountTasksRequest) (*CountTasksResponse, error) {
	return p.cc.CountTasks(ctx, in)
}
