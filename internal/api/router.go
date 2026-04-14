package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/api/handlers"
	"github.com/onetap/salary-advance-loan-service/internal/api/middleware"
	"github.com/onetap/salary-advance-loan-service/internal/auth"
	"github.com/onetap/salary-advance-loan-service/internal/ratelimit"
	"github.com/onetap/salary-advance-loan-service/internal/rating"
	"github.com/onetap/salary-advance-loan-service/internal/transaction"
	"github.com/onetap/salary-advance-loan-service/internal/validation"
)

// RouterConfig holds the configuration for the router
type RouterConfig struct {
	AuthService        auth.AuthService
	RateLimiter        ratelimit.RateLimiter
	ValidationService  *validation.ValidationService
	TransactionService transaction.TransactionService
	RatingService      rating.RatingService
}

// SetupRouter creates and configures the Gin router with all middleware and routes
func SetupRouter(config RouterConfig) *gin.Engine {
	// Create Gin router
	router := gin.New()

	// Add global middleware (order matters!)
	router.Use(middleware.Recovery())                       // Panic recovery
	router.Use(middleware.CorrelationID())                  // Correlation ID
	router.Use(middleware.Logger(middleware.LogLevelInfo))  // Request logging
	router.Use(middleware.SecurityHeaders())                // Security headers
	router.Use(middleware.RequestTimeout(30 * time.Second)) // Request timeout
	router.Use(middleware.InputValidation())                // Input validation
	router.Use(middleware.ErrorHandler())                   // Error handling

	// Create handlers
	authHandler := handlers.NewAuthHandler(config.AuthService, config.RateLimiter)
	customerHandler := handlers.NewCustomerHandler(
		config.ValidationService,
		config.TransactionService,
		config.RatingService,
	)
	validationHandler := handlers.NewValidationHandler(config.ValidationService)

	// Public routes (no authentication required)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/health", healthCheck)

	// Protected routes (authentication required)
	authenticated := router.Group("/")
	authenticated.Use(middleware.AuthMiddleware(config.AuthService))
	{
		// Admin and uploader routes
		adminUploader := authenticated.Group("/")
		adminUploader.Use(middleware.RequireRole("admin", "uploader"))
		{
			adminUploader.POST("/customers/validate", customerHandler.ValidateCustomers)
			adminUploader.GET("/customers/:accountNo/transactions", customerHandler.GetCustomerTransactions)
			adminUploader.GET("/customers/:accountNo/rating", customerHandler.GetCustomerRating)
		}

		// Admin-only routes
		adminOnly := authenticated.Group("/")
		adminOnly.Use(middleware.RequireRole("admin"))
		{
			adminOnly.GET("/validation/logs", validationHandler.GetValidationLogs)
			adminOnly.GET("/validation/processed-records", validationHandler.GetProcessedRecords)
		}
	}

	return router
}

// healthCheck handles GET /health
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
