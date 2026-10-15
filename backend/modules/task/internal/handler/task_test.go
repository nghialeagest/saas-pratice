package handler_test

import (
	"os"

	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"saasbase/platform/authz"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	noteapi "saasbase/modules/note/api"
	"saasbase/modules/task/internal/biz"
	"saasbase/modules/task/internal/handler"
	"saasbase/modules/task/internal/service"
)

type mockTaskRepo struct {
	biz.Repo
}

func (m *mockTaskRepo) Create(ctx context.Context, t *biz.Task) error {
	t.ID = "task-456"
	t.CreatedAt = time.Now()
	return nil
}

func (m *mockTaskRepo) Get(ctx context.Context, id string) (*biz.Task, error) {
	return &biz.Task{
		ID:        id,
		Title:     "Sample Task",
		NoteID:    "note-123",
		NoteTitle: "Sample Note",
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockTaskRepo) Count(ctx context.Context) (int64, error) {
	return 17, nil
}

type mockNoteService struct {
	noteapi.NoteServiceServer
}

func (m *mockNoteService) GetNote(ctx context.Context, req *noteapi.GetNoteRequest) (*noteapi.Note, error) {
	return &noteapi.Note{
		Id:    req.GetId(),
		Title: "Sample Note",
	}, nil
}

func TestMain(m *testing.M) {
	authz.SetDefaultChecker(authz.AllowAllChecker{})
	os.Exit(m.Run())
}

func TestTaskHandler_DTO_Serialization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.New(&mockTaskRepo{}, &mockNoteService{})
	h := handler.New(svc)

	r := gin.New()
	h.Register(r)

	// 1. Test Create Task -> verify clean TaskResponse DTO
	createReq := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(`{"title":"New Task","note_id":"note-123"}`))
	createReq.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, createReq)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	bodyStr := wCreate.Body.String()
	// Assert no proto internals in JSON
	if strings.Contains(bodyStr, "state") || strings.Contains(bodyStr, "sizeCache") || strings.Contains(bodyStr, "unknownFields") {
		t.Fatalf("response contains protobuf internal fields: %s", bodyStr)
	}

	var taskDTO handler.TaskResponse
	if err := json.Unmarshal(wCreate.Body.Bytes(), &taskDTO); err != nil {
		t.Fatalf("failed to unmarshal into TaskResponse DTO: %v", err)
	}
	if taskDTO.ID != "task-456" || taskDTO.Title != "New Task" || taskDTO.NoteID != "note-123" || taskDTO.NoteTitle != "Sample Note" {
		t.Fatalf("unexpected TaskResponse DTO content: %+v", taskDTO)
	}

	// 2. Test Get Task -> verify TaskResponse DTO
	getReq := httptest.NewRequest(http.MethodGet, "/v1/tasks/task-456", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wGet.Code, wGet.Body.String())
	}

	var gotTaskDTO handler.TaskResponse
	if err := json.Unmarshal(wGet.Body.Bytes(), &gotTaskDTO); err != nil {
		t.Fatalf("failed to unmarshal into TaskResponse DTO: %v", err)
	}
	if gotTaskDTO.ID != "task-456" || gotTaskDTO.Title != "Sample Task" {
		t.Fatalf("unexpected GetTask content: %+v", gotTaskDTO)
	}

	// 3. Test Count Tasks -> verify CountTasksResponse DTO
	countReq := httptest.NewRequest(http.MethodGet, "/v1/tasks/count", nil)
	wCount := httptest.NewRecorder()
	r.ServeHTTP(wCount, countReq)

	if wCount.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wCount.Code, wCount.Body.String())
	}

	var countDTO handler.CountTasksResponse
	if err := json.Unmarshal(wCount.Body.Bytes(), &countDTO); err != nil {
		t.Fatalf("failed to unmarshal into CountTasksResponse DTO: %v", err)
	}
	if countDTO.Count != 17 {
		t.Fatalf("expected count 17, got %d", countDTO.Count)
	}
}
