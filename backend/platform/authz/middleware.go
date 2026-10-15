package authz

import (
	"context"
	stderrors "errors"
	"sync"

	"github.com/gin-gonic/gin"

	"saasbase/platform/errors"
	"saasbase/platform/httpx"
)

var (
	defaultCheckerMu sync.RWMutex
	defaultChecker   Checker

	errNoChecker = stderrors.New("authz: no Checker configured")
)

// SetDefaultChecker sets the global default Checker used by Require.
func SetDefaultChecker(c Checker) {
	defaultCheckerMu.Lock()
	defer defaultCheckerMu.Unlock()
	defaultChecker = c
}

// GetDefaultChecker retrieves the current global default Checker.
func GetDefaultChecker() Checker {
	defaultCheckerMu.RLock()
	defer defaultCheckerMu.RUnlock()
	return defaultChecker
}

// Require returns a Gin middleware that enforces the specified namespace:action permission.
func Require(namespace, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		checker := GetDefaultChecker()
		if checker == nil {
			httpx.Fail(c, errors.Internal(errNoChecker))
			c.Abort()
			return
		}

		if err := checker.Check(c.Request.Context(), namespace, action); err != nil {
			httpx.Fail(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWith creates an authorization middleware using a specific Checker instance.
func RequireWith(checker Checker, namespace, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			httpx.Fail(c, errors.Internal(errNoChecker))
			c.Abort()
			return
		}
		if err := checker.Check(c.Request.Context(), namespace, action); err != nil {
			httpx.Fail(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

// MustHaveChecker panics when no default Checker is registered. Call it during
// startup so a misconfigured container fails immediately instead of serving
// unprotected routes.
func MustHaveChecker() {
	if GetDefaultChecker() == nil {
		panic("authz: no default Checker registered")
	}
}

// AllowAllChecker grants every permission. It exists for tests and local
// harnesses so that production code can keep failing closed.
type AllowAllChecker struct{}

var _ Checker = AllowAllChecker{}

func (AllowAllChecker) Check(context.Context, string, string) error { return nil }
