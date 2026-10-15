package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	platformerrors "saasbase/platform/errors"
	"saasbase/platform/httpx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestFail_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	appErr := platformerrors.NotFound("item.not_found", "item does not exist")
	httpx.Fail(c, appErr)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp["code"] != "item.not_found" {
		t.Errorf("expected code 'item.not_found', got %q", resp["code"])
	}
	if resp["message"] != "item does not exist" {
		t.Errorf("expected message 'item does not exist', got %q", resp["message"])
	}
}

func TestFail_AppErrorInternal(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	orig := errors.New("sensitive db secret connection error")
	appErr := platformerrors.Internal(orig)
	httpx.Fail(c, appErr)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp["code"] != platformerrors.CodeInternal {
		t.Errorf("expected code 'internal', got %q", resp["code"])
	}
	if resp["message"] != "internal server error" {
		t.Errorf("expected message 'internal server error', got %q", resp["message"])
	}
}

func TestFail_GormErrRecordNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httpx.Fail(c, gorm.ErrRecordNotFound)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp["code"] != "not_found" {
		t.Errorf("expected code 'not_found', got %q", resp["code"])
	}
	if resp["message"] != "not found" {
		t.Errorf("expected message 'not found', got %q", resp["message"])
	}
}

func TestFail_DefaultRawError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httpx.Fail(c, errors.New("unhandled fatal crash"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp["code"] != platformerrors.CodeInternal {
		t.Errorf("expected code 'internal', got %q", resp["code"])
	}
	if resp["message"] != "internal server error" {
		t.Errorf("expected message 'internal server error', got %q", resp["message"])
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httpx.BadRequest(c, errors.New("invalid payload"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp["error"] != "invalid payload" {
		t.Errorf("expected error 'invalid payload', got %q", resp["error"])
	}
}
