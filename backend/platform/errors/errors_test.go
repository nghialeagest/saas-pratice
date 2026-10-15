package errors_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"

	platformerrors "saasbase/platform/errors"
)

func TestAppErrorInterface(t *testing.T) {
	var _ error = (*platformerrors.AppError)(nil)

	errWithMsg := platformerrors.NotFound("not_found", "item not found")
	if errWithMsg.Error() != "item not found" {
		t.Fatalf("expected 'item not found', got %q", errWithMsg.Error())
	}

	errNoMsg := &platformerrors.AppError{
		Code: "custom.err",
		Err:  io.EOF,
	}
	if errNoMsg.Error() != "EOF" {
		t.Fatalf("expected 'EOF', got %q", errNoMsg.Error())
	}

	errOnlyCode := &platformerrors.AppError{
		Code: "custom.err",
	}
	if errOnlyCode.Error() != "custom.err" {
		t.Fatalf("expected 'custom.err', got %q", errOnlyCode.Error())
	}

	var nilErr *platformerrors.AppError
	if nilErr.Error() != "" {
		t.Fatalf("expected empty string for nil error, got %q", nilErr.Error())
	}
}

func TestErrorsAs(t *testing.T) {
	appErr := platformerrors.BadRequest("invalid_input", "bad field")
	wrapped := fmt.Errorf("service layer: %w", appErr)

	var extracted *platformerrors.AppError
	if !errors.As(wrapped, &extracted) {
		t.Fatal("expected errors.As to extract *AppError")
	}

	if extracted.Code != "invalid_input" {
		t.Errorf("expected code 'invalid_input', got %q", extracted.Code)
	}
	if extracted.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, extracted.HTTPStatus)
	}
	if extracted.Message != "bad field" {
		t.Errorf("expected message 'bad field', got %q", extracted.Message)
	}
}

func TestErrorsIs(t *testing.T) {
	appErr := platformerrors.NotFound("note.not_found", "note not found")

	// Match any AppError when Code is empty
	if !errors.Is(appErr, &platformerrors.AppError{}) {
		t.Error("expected errors.Is to match empty AppError")
	}

	// Match by code
	if !errors.Is(appErr, &platformerrors.AppError{Code: "note.not_found"}) {
		t.Error("expected errors.Is to match AppError with same code")
	}

	// Do not match different code
	if errors.Is(appErr, &platformerrors.AppError{Code: "task.not_found"}) {
		t.Error("expected errors.Is not to match AppError with different code")
	}

	// Test with builder function
	matchBuilder := platformerrors.NotFound("note.not_found", "different message")
	if !errors.Is(appErr, matchBuilder) {
		t.Error("expected errors.Is to match AppError created by builder with same code")
	}

	// Unwrap underlying error
	underlying := errors.New("underlying error")
	internalErr := platformerrors.Internal(underlying)
	if !errors.Is(internalErr, underlying) {
		t.Error("expected errors.Is to match underlying error via Unwrap")
	}
}

func TestInternalHidesOriginalError(t *testing.T) {
	orig := errors.New("sensitive database connection failed on port 5432")
	appErr := platformerrors.Internal(orig)

	if appErr.Error() == orig.Error() {
		t.Fatalf("expected original error message to be hidden, got %q", appErr.Error())
	}
	if appErr.Error() != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", appErr.Error())
	}
	if appErr.Message != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", appErr.Message)
	}
	if appErr.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", appErr.HTTPStatus)
	}
	if appErr.Code != platformerrors.CodeInternal {
		t.Errorf("expected code %q, got %q", platformerrors.CodeInternal, appErr.Code)
	}
	if appErr.Err != orig {
		t.Errorf("expected underlying Err to be preserved")
	}
	if !errors.Is(appErr, orig) {
		t.Errorf("expected errors.Is(appErr, orig) to be true")
	}
}

func TestBuilderFunctionsHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        *platformerrors.AppError
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "NotFound",
			err:        platformerrors.NotFound("note.not_found", "note not found"),
			wantStatus: http.StatusNotFound,
			wantCode:   "note.not_found",
			wantMsg:    "note not found",
		},
		{
			name:       "BadRequest",
			err:        platformerrors.BadRequest("invalid_arg", "bad argument"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_arg",
			wantMsg:    "bad argument",
		},
		{
			name:       "Unauthorized",
			err:        platformerrors.Unauthorized("missing auth token"),
			wantStatus: http.StatusUnauthorized,
			wantCode:   platformerrors.CodeUnauthorized,
			wantMsg:    "missing auth token",
		},
		{
			name:       "UnauthorizedDefaultMsg",
			err:        platformerrors.Unauthorized(""),
			wantStatus: http.StatusUnauthorized,
			wantCode:   platformerrors.CodeUnauthorized,
			wantMsg:    "unauthorized",
		},
		{
			name:       "Forbidden",
			err:        platformerrors.Forbidden("tenant.forbidden", "tenant access forbidden"),
			wantStatus: http.StatusForbidden,
			wantCode:   "tenant.forbidden",
			wantMsg:    "tenant access forbidden",
		},
		{
			name:       "Internal",
			err:        platformerrors.Internal(errors.New("db error")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   platformerrors.CodeInternal,
			wantMsg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.wantStatus {
				t.Errorf("HTTPStatus = %d, want %d", tt.err.HTTPStatus, tt.wantStatus)
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
			if tt.err.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.wantMsg)
			}
		})
	}
}

func TestGRPCCodeMapping(t *testing.T) {
	tests := []struct {
		httpStatus int
		wantCode   codes.Code
	}{
		{http.StatusOK, codes.OK},
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusUnauthorized, codes.Unauthenticated},
		{http.StatusForbidden, codes.PermissionDenied},
		{http.StatusNotFound, codes.NotFound},
		{http.StatusConflict, codes.AlreadyExists},
		{http.StatusTooManyRequests, codes.ResourceExhausted},
		{http.StatusNotImplemented, codes.Unimplemented},
		{http.StatusServiceUnavailable, codes.Unavailable},
		{http.StatusGatewayTimeout, codes.DeadlineExceeded},
		{http.StatusRequestTimeout, codes.DeadlineExceeded},
		{http.StatusInternalServerError, codes.Internal},
		{599, codes.Internal},
	}

	for _, tt := range tests {
		err := &platformerrors.AppError{HTTPStatus: tt.httpStatus}
		if got := err.GRPCCode(); got != tt.wantCode {
			t.Errorf("GRPCCode() for HTTP status %d = %v, want %v", tt.httpStatus, got, tt.wantCode)
		}
	}

	var nilErr *platformerrors.AppError
	if got := nilErr.GRPCCode(); got != codes.OK {
		t.Errorf("GRPCCode() for nil error = %v, want %v", got, codes.OK)
	}
}
