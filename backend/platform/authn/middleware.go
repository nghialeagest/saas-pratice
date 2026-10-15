package authn

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"saasbase/platform/errors"
	"saasbase/platform/httpx"
)

// SessionValidator validates whether a session is alive and not revoked.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID string) bool
}

// SessionValidatorFunc allows using a plain function as a SessionValidator.
type SessionValidatorFunc func(ctx context.Context, sessionID string) bool

func (f SessionValidatorFunc) ValidateSession(ctx context.Context, sessionID string) bool {
	return f(ctx, sessionID)
}

type MiddlewareOptions struct {
	TokenConfig      TokenConfig
	SessionValidator SessionValidator
	PublicPrefixes   []string
}

// GinMiddleware enforces authentication on all routes by default (fail-closed).
// Only explicitly registered PublicPrefixes bypass authentication.
func GinMiddleware(opts MiddlewareOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		isPublic := false
		for _, prefix := range opts.PublicPrefixes {
			if strings.HasPrefix(path, prefix) {
				isPublic = true
				break
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if isPublic {
				c.Next()
				return
			}
			httpx.Fail(c, errors.Unauthorized("authorization header required"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			if isPublic {
				c.Next()
				return
			}
			httpx.Fail(c, errors.Unauthorized("invalid authorization format, expected Bearer <token>"))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := ParseAccess(opts.TokenConfig, tokenStr)
		if err != nil {
			httpx.Fail(c, errors.Unauthorized("invalid or expired token"))
			c.Abort()
			return
		}

		// Verify session is active in DB (session guard)
		if opts.SessionValidator != nil && claims.SessionID != "" {
			if !opts.SessionValidator.ValidateSession(c.Request.Context(), claims.SessionID) {
				httpx.Fail(c, errors.Unauthorized("session revoked or expired"))
				c.Abort()
				return
			}
		}

		// Inject Identity into context
		id := Identity{
			UserID:    claims.UserID,
			TenantID:  claims.TenantID,
			SessionID: claims.SessionID,
		}
		c.Request = c.Request.WithContext(NewContext(c.Request.Context(), id))
		c.Next()
	}
}

// GRPCUnaryInterceptor enforces authentication on gRPC calls.
func GRPCUnaryInterceptor(cfg TokenConfig, validator SessionValidator, publicMethods ...string) grpc.UnaryServerInterceptor {
	publicMap := make(map[string]bool)
	for _, m := range publicMethods {
		publicMap[m] = true
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info != nil && publicMap[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata missing")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization metadata missing")
		}

		parts := strings.SplitN(tokens[0], " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
		}

		claims, err := ParseAccess(cfg, parts[1])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		if validator != nil && claims.SessionID != "" {
			if !validator.ValidateSession(ctx, claims.SessionID) {
				return nil, status.Error(codes.Unauthenticated, "session revoked or expired")
			}
		}

		id := Identity{
			UserID:    claims.UserID,
			TenantID:  claims.TenantID,
			SessionID: claims.SessionID,
		}
		newCtx := NewContext(ctx, id)
		return handler(newCtx, req)
	}
}
