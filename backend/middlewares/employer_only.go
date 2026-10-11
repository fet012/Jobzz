package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)



// EmployerOnly must be used AFTER Auth().
// It aborts with 403 if the authenticated user is not an employer.
func EmployerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(CtxRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}

		if role != "employer" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Employer access required"})
			return
		}

		c.Next()
	}
}