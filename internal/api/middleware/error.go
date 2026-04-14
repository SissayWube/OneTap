package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apierrors "github.com/onetap/salary-advance-loan-service/internal/api/errors"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		correlationID, _ := c.Get("correlation_id")

		resp := gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "An unexpected error occurred",
			},
			"meta": gin.H{
				"request_id": correlationID,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
		}
		status := http.StatusInternalServerError

		if appErr, ok := err.(apierrors.AppError); ok {
			errBody := gin.H{
				"code":    appErr.Code(),
				"message": appErr.Error(),
			}
			if details := appErr.Details(); details != nil {
				errBody["details"] = details
			}
			resp["error"] = errBody
			status = appErr.HTTPStatus()
		}

		c.JSON(status, resp)
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				correlationID, _ := c.Get("correlation_id")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An unexpected error occurred",
					},
					"meta": gin.H{
						"request_id": correlationID,
						"timestamp":  time.Now().UTC().Format(time.RFC3339),
					},
				})
			}
		}()
		c.Next()
	}
}
