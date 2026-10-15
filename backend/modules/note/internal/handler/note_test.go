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

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"saasbase/modules/note/internal/biz"
	"saasbase/modules/note/internal/handler"
	"saasbase/modules/note/internal/service"
)

type mockRepo struct {
	biz.Repo
}

func (m *mockRepo) Get(ctx context.Context, id string) (*biz.Note, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestMain(m *testing.M) {
	authz.SetDefaultChecker(authz.AllowAllChecker{})
	os.Exit(m.Run())
}

func TestGetNote_NotFound_Returns404WithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.New(&mockRepo{})
	h := handler.New(svc)

	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/notes/unknown-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "note.not_found" {
		t.Fatalf("expected code 'note.not_found', got %q", resp.Code)
	}
}

type successMockRepo struct {
	biz.Repo
}

func (s *successMockRepo) Create(ctx context.Context, n *biz.Note) error {
	n.ID = "note-123"
	n.Status = biz.StatusDraft
	return nil
}

func (s *successMockRepo) Get(ctx context.Context, id string) (*biz.Note, error) {
	return &biz.Note{
		ID:     id,
		Title:  "Test Note",
		Status: biz.StatusPublished,
	}, nil
}

func (s *successMockRepo) List(ctx context.Context, q biz.ListQuery) ([]*biz.Note, error) {
	return []*biz.Note{
		{ID: "n1", Title: "Note 1", Status: biz.StatusDraft},
		{ID: "n2", Title: "Note 2", Status: biz.StatusPublished},
	}, nil
}

func (s *successMockRepo) Count(ctx context.Context, status string) (int64, error) {
	return 42, nil
}

func TestNoteHandler_DTO_Serialization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.New(&successMockRepo{})
	h := handler.New(svc)

	r := gin.New()
	h.Register(r)

	// 1. Test Create note -> verifies clean JSON DTO
	createReq := httptest.NewRequest(http.MethodPost, "/v1/notes", strings.NewReader(`{"title":"My Note"}`))
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

	var noteDTO handler.NoteResponse
	if err := json.Unmarshal(wCreate.Body.Bytes(), &noteDTO); err != nil {
		t.Fatalf("failed to unmarshal into NoteResponse DTO: %v", err)
	}
	if noteDTO.ID != "note-123" || noteDTO.Title != "My Note" || noteDTO.Status != "DRAFT" {
		t.Fatalf("unexpected NoteResponse DTO content: %+v", noteDTO)
	}

	// 2. Test List notes -> verifies ListNotesResponse DTO
	listReq := httptest.NewRequest(http.MethodGet, "/v1/notes", nil)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, listReq)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wList.Code, wList.Body.String())
	}

	var listDTO handler.ListNotesResponse
	if err := json.Unmarshal(wList.Body.Bytes(), &listDTO); err != nil {
		t.Fatalf("failed to unmarshal into ListNotesResponse DTO: %v", err)
	}
	if len(listDTO.Items) != 2 || listDTO.Total != 42 {
		t.Fatalf("unexpected ListNotesResponse content: %+v", listDTO)
	}

	// 3. Test Count notes -> verifies CountNotesResponse DTO
	countReq := httptest.NewRequest(http.MethodGet, "/v1/notes/count", nil)
	wCount := httptest.NewRecorder()
	r.ServeHTTP(wCount, countReq)

	if wCount.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wCount.Code, wCount.Body.String())
	}

	var countDTO handler.CountNotesResponse
	if err := json.Unmarshal(wCount.Body.Bytes(), &countDTO); err != nil {
		t.Fatalf("failed to unmarshal into CountNotesResponse DTO: %v", err)
	}
	if countDTO.Count != 42 {
		t.Fatalf("expected count 42, got %d", countDTO.Count)
	}
}
