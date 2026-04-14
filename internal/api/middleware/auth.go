package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/auth"
)

func AuthMiddleware(authService auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "MISSING_TOKEN",
				"message": "Authorization header is required",
			}})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "INVALID_TOKEN_FORMAT",
				"message": "Authorization header must be in format: Bearer <token>",
			}})
			return
		}

		claims, err := authService.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid or expired token",
			}})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "MISSING_AUTHENTICATION",
				"message": "Authentication required",
			}})
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Invalid role format",
			}})
			return
		}

		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "INSUFFICIENT_PERMISSIONS",
			"message": "You do not have permission to access this resource",
		}})
	}
}
