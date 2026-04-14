package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/rating"
	"github.com/onetap/salary-advance-loan-service/internal/transaction"
	"github.com/onetap/salary-advance-loan-service/internal/validation"
)

type CustomerHandler struct {
	validationService  *validation.ValidationService
	transactionService transaction.TransactionService
	ratingService      rating.RatingService
}

func NewCustomerHandler(
	validationService *validation.ValidationService,
	transactionService transaction.TransactionService,
	ratingService rating.RatingService,
) *CustomerHandler {
	return &CustomerHandler{
		validationService:  validationService,
		transactionService: transactionService,
		ratingService:      ratingService,
	}
}

func (h *CustomerHandler) ValidateCustomers(c *gin.Context) {
	var req struct {
		Records []models.CustomerRecord `json:"records"`
	}

	ctx := c.Request.Context()

	if err := c.ShouldBindJSON(&req); err == nil && len(req.Records) > 0 {
		c.JSON(http.StatusOK, h.validationService.ProcessSampleCustomers(ctx, req.Records))
		return
	}

	result, err := h.validationService.ProcessConfiguredSampleCustomers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to process sample customers",
			"details": err.Error(),
		}})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CustomerHandler) GetCustomerTransactions(c *gin.Context) {
	accountNo := c.Param("accountNo")
	if accountNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Account number is required",
		}})
		return
	}

	txs, err := h.transactionService.GetCustomerTransactions(c.Request.Context(), accountNo)
	if err != nil {
		if err == transaction.ErrCustomerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "RESOURCE_NOT_FOUND",
				"message": "Customer not found",
			}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to retrieve transactions",
		}})
		return
	}

	stats, err := h.transactionService.CalculateTransactionStats(c.Request.Context(), accountNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to calculate transaction statistics",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_no":   accountNo,
		"transactions": txs,
		"stats":        stats,
	})
}

func (h *CustomerHandler) GetCustomerRating(c *gin.Context) {
	accountNo := c.Param("accountNo")
	if accountNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Account number is required",
		}})
		return
	}

	r, err := h.ratingService.CalculateRating(c.Request.Context(), accountNo)
	if err != nil {
		if err == rating.ErrCustomerNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "RESOURCE_NOT_FOUND",
				"message": "Customer not found",
			}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to calculate rating",
		}})
		return
	}

	c.JSON(http.StatusOK, r)
}
