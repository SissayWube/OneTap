package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/validation"
)

// ValidationHandler handles validation-related requests
type ValidationHandler struct {
	validationService *validation.ValidationService
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(validationService *validation.ValidationService) *ValidationHandler {
	return &ValidationHandler{
		validationService: validationService,
	}
}

// GetValidationLogs handles GET /validation/logs
func (h *ValidationHandler) GetValidationLogs(c *gin.Context) {
	// Parse query parameters
	var filters models.ValidationFilters

	// Parse batch_number (optional)
	if batchNumStr := c.Query("batch_number"); batchNumStr != "" {
		batchNum, err := strconv.Atoi(batchNumStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid batch_number parameter",
					"details": "batch_number must be an integer",
				},
			})
			return
		}
		filters.BatchNumber = &batchNum
	}

	// Parse verified (optional)
	if verifiedStr := c.Query("verified"); verifiedStr != "" {
		verified, err := strconv.ParseBool(verifiedStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid verified parameter",
					"details": "verified must be a boolean (true/false)",
				},
			})
			return
		}
		filters.Verified = &verified
	}

	// Parse limit (optional, default: 50)
	filters.Limit = 50
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid limit parameter",
					"details": "limit must be an integer between 1 and 1000",
				},
			})
			return
		}
		filters.Limit = limit
	}

	// Parse offset (optional, default: 0)
	filters.Offset = 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid offset parameter",
					"details": "offset must be a non-negative integer",
				},
			})
			return
		}
		filters.Offset = offset
	}

	// Get validation logs
	total, err := h.validationService.CountValidationLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to count validation logs",
			},
		})
		return
	}

	logs, err := h.validationService.GetValidationLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve validation logs",
			},
		})
		return
	}

	// Return logs with pagination metadata
	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"total":  total,
		"limit":  filters.Limit,
		"offset": filters.Offset,
	})
}

// GetProcessedRecords handles GET /validation/processed-records
func (h *ValidationHandler) GetProcessedRecords(c *gin.Context) {
	records, err := h.validationService.GetProcessedRecords(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve processed records",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   len(records),
	})
}
