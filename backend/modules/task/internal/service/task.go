package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	noteapi "saasbase/modules/note/api"
	"saasbase/modules/task/api"
	"saasbase/modules/task/internal/biz"
	"saasbase/platform/errors"
)

type Service struct {
	repo  biz.Repo
	notes noteapi.NoteServiceServer
}

var _ api.TaskServiceServer = (*Service)(nil)

func New(repo biz.Repo, notes noteapi.NoteServiceServer) *Service {
	return &Service{repo: repo, notes: notes}
}

func (s *Service) CreateTask(ctx context.Context, req *api.CreateTaskRequest) (*api.Task, error) {
	if req.GetTitle() == "" {
		return nil, errors.New("title is required")
	}

	e := &biz.Task{
		ID:    uuid.NewString(),
		Title: req.GetTitle(),
	}

	if id := req.GetNoteId(); id != "" {
		note, err := s.notes.GetNote(ctx, &noteapi.GetNoteRequest{Id: id})
		if err != nil {
			return nil, err
		}
		e.NoteID = note.GetId()
		e.NoteTitle = note.GetTitle()
	}

	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return toProto(e), nil
}

func (s *Service) GetTask(ctx context.Context, req *api.GetTaskRequest) (*api.Task, error) {
	e, err := s.repo.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("task.not_found", "task not found")
		}
		return nil, err
	}
	return toProto(e), nil
}

func (s *Service) CountTasks(ctx context.Context, _ *api.CountTasksRequest) (*api.CountTasksResponse, error) {
	n, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &api.CountTasksResponse{Count: n}, nil
}

func toProto(e *biz.Task) *api.Task {
	return &api.Task{
		Id:        e.ID,
		Title:     e.Title,
		NoteId:    e.NoteID,
		NoteTitle: e.NoteTitle,
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
}
