package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"jobz/utils"
)

// Context keys. Using constants instead of raw strings prevents typos
// (e.g. "userId" vs "userid") that would silently return empty values.
const (
	CtxUserID = "userID"
	CtxRole   = "role"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be 'Bearer <token>'"})
			return
		}

		claims, err := utils.VerifyToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)

		c.Next()
	}
}