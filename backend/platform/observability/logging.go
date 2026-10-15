package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"saasbase/platform/authn"
	"saasbase/platform/tenant"
)

var sensitiveKeys = []string{
	"password",
	"passwd",
	"token",
	"secret",
	"hash",
	"authorization",
	"auth_header",
	"access_token",
	"refresh_token",
}

// sanitizeAttr inspects and masks sensitive keys or values to avoid leaking secrets.
func sanitizeAttr(groups []string, a slog.Attr) slog.Attr {
	keyLower := strings.ToLower(a.Key)
	for _, sk := range sensitiveKeys {
		if strings.Contains(keyLower, sk) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}

	// Scrub strings that look like Bearer tokens or bcrypt hashes
	if a.Value.Kind() == slog.KindString {
		strVal := a.Value.String()
		if strings.HasPrefix(strVal, "Bearer ") {
			return slog.String(a.Key, "Bearer [REDACTED]")
		}
		if strings.HasPrefix(strVal, "$2a$") || strings.HasPrefix(strVal, "$2b$") {
			return slog.String(a.Key, "[REDACTED_HASH]")
		}
	}

	return a
}

// ContextHandler injects request_id, user_id, and tenant_id into every log record.
type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID := RequestIDFromContext(ctx); reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
		if id, ok := authn.FromContext(ctx); ok {
			if id.UserID != "" {
				r.AddAttrs(slog.String("user_id", id.UserID))
			}
			if id.TenantID != "" {
				r.AddAttrs(slog.String("tenant_id", id.TenantID))
			}
		}
		if tid := tenant.FromContext(ctx); tid != "" {
			// Ensure tenant_id is set if not already from authn
			r.AddAttrs(slog.String("tenant_id", tid))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// NewLogger creates an slog.Logger with context enrichment and secret sanitization.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	baseHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: sanitizeAttr,
	})
	return slog.New(&ContextHandler{Handler: baseHandler})
}

// HTTPLoggingMiddleware logs HTTP request completions with request context attributes.
func HTTPLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		ctx := c.Request.Context()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		logger.Log(ctx, level, "http request",
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
		)
	}
}
