package errors

import (
	stderrors "errors"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
)

// Error code constants as strings.
const (
	CodeNoteNotFound    = "note.not_found"
	CodeTaskNotFound    = "task.not_found"
	CodeUserNotFound    = "user.not_found"
	CodeTenantForbidden = "tenant.forbidden"
	CodeUnauthorized    = "unauthorized"
	CodeInternal        = "internal"

	// Aliases for convenience.
	NoteNotFound    = CodeNoteNotFound
	TaskNotFound    = CodeTaskNotFound
	UserNotFound    = CodeUserNotFound
	TenantForbidden = CodeTenantForbidden
)

// AppError represents an application error with machine-readable code,
// HTTP status code, user-safe message, and optional underlying error.
type AppError struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"-"`
	Message    string `json:"message"`
	Err        error  `json:"-"`
}

// Error implements the error interface. It returns Message if non-empty,
// or Err.Error() if no message is set, or Code if both are empty.
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is reports whether target matches this AppError.
// It matches if target is an AppError with an empty Code or matching Code.
func (e *AppError) Is(target error) bool {
	if e == nil {
		return false
	}
	t, ok := target.(*AppError)
	if !ok || t == nil {
		return false
	}
	return t.Code == "" || e.Code == t.Code
}

// GRPCCode maps the HTTPStatus to a corresponding gRPC status code.
func (e *AppError) GRPCCode() codes.Code {
	if e == nil {
		return codes.OK
	}
	switch e.HTTPStatus {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}

// WithError returns a shallow copy of the AppError with Err set to err.
func (e *AppError) WithError(err error) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.Err = err
	return &clone
}

// NotFound creates a new 404 AppError.
func NotFound(code, msg string) *AppError {
	return &AppError{
		Code:       code,
		HTTPStatus: http.StatusNotFound,
		Message:    msg,
	}
}

// BadRequest creates a new 400 AppError.
func BadRequest(code, msg string) *AppError {
	return &AppError{
		Code:       code,
		HTTPStatus: http.StatusBadRequest,
		Message:    msg,
	}
}

// Unauthorized creates a new 401 AppError.
func Unauthorized(msg string) *AppError {
	if msg == "" {
		msg = "unauthorized"
	}
	return &AppError{
		Code:       CodeUnauthorized,
		HTTPStatus: http.StatusUnauthorized,
		Message:    msg,
	}
}

// Forbidden creates a new 403 AppError.
func Forbidden(code, msg string) *AppError {
	return &AppError{
		Code:       code,
		HTTPStatus: http.StatusForbidden,
		Message:    msg,
	}
}

// Internal creates a new 500 AppError that hides the original error message
// and uses a generic user-facing message.
func Internal(err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "internal server error",
		Err:        err,
	}
}

// Re-exports of standard library errors functions so callers can import
// this package as a drop-in replacement.
func New(text string) error {
	return stderrors.New(text)
}

func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

func As(err error, target any) bool {
	return stderrors.As(err, target)
}

func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Conflict creates a new 409 AppError, for unique-constraint style failures.
func Conflict(code, msg string) *AppError {
	return &AppError{
		Code:       code,
		HTTPStatus: http.StatusConflict,
		Message:    msg,
	}
}

// IsDuplicateKey reports whether err is a unique-constraint violation.
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "sqlstate 23505")
}
