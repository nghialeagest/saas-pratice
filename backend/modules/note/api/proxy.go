package api

import (
	"context"

	"google.golang.org/grpc"
)

type clientProxy struct {
	cc NoteServiceClient
}

var _ NoteServiceServer = (*clientProxy)(nil)

// NewClientProxy adapts a gRPC client to the server interface so callers can
// depend on NoteServiceServer regardless of where the module runs.
func NewClientProxy(cc NoteServiceClient) NoteServiceServer {
	return &clientProxy{cc: cc}
}

func NewClientProxyFromConn(conn grpc.ClientConnInterface) NoteServiceServer {
	return NewClientProxy(NewNoteServiceClient(conn))
}

func (p *clientProxy) CreateNote(ctx context.Context, in *CreateNoteRequest) (*Note, error) {
	return p.cc.CreateNote(ctx, in)
}

func (p *clientProxy) GetNote(ctx context.Context, in *GetNoteRequest) (*Note, error) {
	return p.cc.GetNote(ctx, in)
}

func (p *clientProxy) ListNotes(ctx context.Context, in *ListNotesRequest) (*ListNotesResponse, error) {
	return p.cc.ListNotes(ctx, in)
}

func (p *clientProxy) CountNotes(ctx context.Context, in *CountNotesRequest) (*CountNotesResponse, error) {
	return p.cc.CountNotes(ctx, in)
}

func (p *clientProxy) PublishNote(ctx context.Context, in *PublishNoteRequest) (*Note, error) {
	return p.cc.PublishNote(ctx, in)
}
