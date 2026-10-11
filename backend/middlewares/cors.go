package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS allows browser requests from the frontend origins.
// Allowed origins come from FRONTEND_ORIGINS (comma-separated) plus localhost dev URLs.
func CORS() gin.HandlerFunc {
	allowed := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}

	if env := os.Getenv("FRONTEND_ORIGINS"); env != "" {
		for _, o := range strings.Split(env, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowed = append(allowed, trimmed)
			}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, o := range allowed {
			if o == origin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}