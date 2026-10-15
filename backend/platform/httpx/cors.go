package httpx

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSOptions defines the configuration options for CORS middleware.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowAllOrigins  bool
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

// DefaultCORSOptions returns sensible default CORS settings for local dev and standard SaaS clients.
func DefaultCORSOptions() CORSOptions {
	defaultOrigins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:8080",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:8080",
	}

	// Override from environment variable CORS_ALLOWED_ORIGINS if provided
	if envOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); envOrigins != "" {
		if envOrigins == "*" {
			return CORSOptions{
				AllowAllOrigins:  true,
				AllowCredentials: false, // Disallow credentials when wild-carding
				AllowedMethods: []string{
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodHead,
					http.MethodOptions,
				},
				AllowedHeaders: []string{
					"Origin",
					"Content-Type",
					"Accept",
					"Authorization",
					"X-Tenant-ID",
					"X-Request-ID",
					"X-Requested-With",
				},
				ExposedHeaders: []string{
					"Content-Length",
					"X-Request-ID",
				},
				MaxAgeSeconds: 86400,
			}
		}

		parts := strings.Split(envOrigins, ",")
		var origins []string
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		if len(origins) > 0 {
			defaultOrigins = origins
		}
	}

	return CORSOptions{
		AllowedOrigins:   defaultOrigins,
		AllowAllOrigins:  false,
		AllowCredentials: true,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Tenant-ID",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},
		MaxAgeSeconds: 86400,
	}
}

// CORSMiddleware returns a Gin middleware enforcing CORS policies.
func CORSMiddleware(opts CORSOptions) gin.HandlerFunc {
	methodsStr := strings.Join(opts.AllowedMethods, ", ")
	headersStr := strings.Join(opts.AllowedHeaders, ", ")
	exposedStr := strings.Join(opts.ExposedHeaders, ", ")

	originsMap := make(map[string]bool)
	for _, o := range opts.AllowedOrigins {
		originsMap[strings.TrimSuffix(o, "/")] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		isAllowed := false
		if opts.AllowAllOrigins {
			isAllowed = true
		} else if origin != "" {
			normalizedOrigin := strings.TrimSuffix(origin, "/")
			if originsMap[normalizedOrigin] {
				isAllowed = true
			}
		}

		if isAllowed {
			if opts.AllowAllOrigins && !opts.AllowCredentials {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				if opts.AllowCredentials {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
			}

			if methodsStr != "" {
				c.Header("Access-Control-Allow-Methods", methodsStr)
			}
			if headersStr != "" {
				c.Header("Access-Control-Allow-Headers", headersStr)
			}
			if exposedStr != "" {
				c.Header("Access-Control-Expose-Headers", exposedStr)
			}
			if opts.MaxAgeSeconds > 0 {
				c.Header("Access-Control-Max-Age", "86400")
			}
		}

		// Handle preflight OPTIONS request immediately to bypass subsequent auth/tenant middleware
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
