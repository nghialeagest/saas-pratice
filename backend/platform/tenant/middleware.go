package tenant

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Validator decides whether the caller may act inside the resolved tenant.
type Validator interface {
	Validate(ctx context.Context, tenantID string) error
}

type Options struct {
	Validator    Validator
	AllowHost    bool
	HostHeaderOK func(c *gin.Context) bool
	PublicPrefixes []string
}

func isPublic(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func Middleware(opt Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublic(c.Request.URL.Path, opt.PublicPrefixes) {
			c.Next()
			return
		}

		id := c.GetHeader(Header)

		if id == "" && !opt.AllowHost {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "missing " + Header,
			})
			return
		}

		ctx := NewContext(c.Request.Context(), id)

		if opt.Validator != nil {
			if err := opt.Validator.Validate(ctx, id); err != nil {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Interceptor propagates the tenant across gRPC calls.
func MetadataKey() string { return "x-tenant-id" }
