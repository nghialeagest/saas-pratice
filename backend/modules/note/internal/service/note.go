package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"saasbase/modules/note/api"
	"saasbase/modules/note/internal/biz"
	"saasbase/platform/errors"
)

type Service struct {
	repo biz.Repo
}

var _ api.NoteServiceServer = (*Service)(nil)

func New(repo biz.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateNote(ctx context.Context, req *api.CreateNoteRequest) (*api.Note, error) {
	if req.GetTitle() == "" {
		return nil, errors.New("title is required")
	}
	e := &biz.Note{
		ID:     uuid.NewString(),
		Title:  req.GetTitle(),
		Body:   req.GetBody(),
		Status: biz.StatusDraft,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return toProto(e), nil
}

func (s *Service) GetNote(ctx context.Context, req *api.GetNoteRequest) (*api.Note, error) {
	e, err := s.repo.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("note.not_found", "note not found")
		}
		return nil, err
	}
	return toProto(e), nil
}

func (s *Service) ListNotes(ctx context.Context, req *api.ListNotesRequest) (*api.ListNotesResponse, error) {
	q := biz.ListQuery{
		Status:   statusToBiz(req.GetStatus()),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	items, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.Count(ctx, q.Status)
	if err != nil {
		return nil, err
	}
	out := make([]*api.Note, len(items))
	for i, e := range items {
		out[i] = toProto(e)
	}
	return &api.ListNotesResponse{Items: out, Total: total}, nil
}

func (s *Service) CountNotes(ctx context.Context, req *api.CountNotesRequest) (*api.CountNotesResponse, error) {
	n, err := s.repo.Count(ctx, statusToBiz(req.GetStatus()))
	if err != nil {
		return nil, err
	}
	return &api.CountNotesResponse{Count: n}, nil
}

func (s *Service) PublishNote(ctx context.Context, req *api.PublishNoteRequest) (*api.Note, error) {
	e, err := s.repo.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("note.not_found", "note not found")
		}
		return nil, err
	}
	e.Publish()
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return toProto(e), nil
}

func toProto(e *biz.Note) *api.Note {
	return &api.Note{
		Id:        e.ID,
		Title:     e.Title,
		Status:    statusToProto(e.Status),
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
}

func statusToProto(s string) api.NoteStatus {
	switch s {
	case biz.StatusDraft:
		return api.NoteStatus_NOTE_STATUS_DRAFT
	case biz.StatusPublished:
		return api.NoteStatus_NOTE_STATUS_PUBLISHED
	default:
		return api.NoteStatus_NOTE_STATUS_UNSPECIFIED
	}
}

func statusToBiz(s api.NoteStatus) string {
	switch s {
	case api.NoteStatus_NOTE_STATUS_DRAFT:
		return biz.StatusDraft
	case api.NoteStatus_NOTE_STATUS_PUBLISHED:
		return biz.StatusPublished
	default:
		return ""
	}
}
