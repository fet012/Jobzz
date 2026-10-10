package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CandidateOnly must be used AFTER Auth().
// It aborts with 403 if the authenticated user is not a candidate.
func CandidateOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(CtxRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}

		if role != "candidate" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Candidate access required"})
			return
		}

		c.Next()
	}
}