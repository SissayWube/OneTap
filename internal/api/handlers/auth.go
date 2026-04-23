package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/auth"
	"github.com/onetap/salary-advance-loan-service/internal/ratelimit"
)

// AuthHandler handles authentication-related HTTP requests.
// It coordinates between the auth service (JWT/password management) and
// rate limiter (brute force protection) to provide secure authentication.
type AuthHandler struct {
	authService auth.AuthService      // Handles JWT generation and validation
	rateLimiter ratelimit.RateLimiter // Tracks and blocks excessive login attempts
}

// NewAuthHandler creates a new authentication handler with the required dependencies.
//
// Parameters:
//   - authService: Service for JWT token generation and validation
//   - rateLimiter: Service for tracking and limiting login attempts
//
// Returns:
//   - *AuthHandler: Configured handler ready to process login requests
func NewAuthHandler(authService auth.AuthService, rateLimiter ratelimit.RateLimiter) *AuthHandler {
	return &AuthHandler{authService: authService, rateLimiter: rateLimiter}
}

// LoginRequest represents the expected JSON payload for login requests.
// Both fields are required and validated by Gin's binding mechanism.
//
// Example JSON:
//
//	{
//	  "username": "admin",
//	  "password": "admin123"
//	}
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // User's login identifier
	Password string `json:"password" binding:"required"` // User's plaintext password (hashed on server)
}

// LoginResponse represents the successful authentication response.
// Contains the JWT token, user information, and token expiration time.
//
// Example JSON:
//
//	{
//	  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//	  "user": {
//	    "id": "admin-001",
//	    "username": "admin",
//	    "role": "admin"
//	  },
//	  "expires_at": "2026-04-23T10:30:00Z"
//	}
type LoginResponse struct {
	Token     string    `json:"token"`      // JWT token for subsequent authenticated requests
	User      UserInfo  `json:"user"`       // User profile information
	ExpiresAt time.Time `json:"expires_at"` // Token expiration timestamp (UTC)
}

// UserInfo contains basic user profile information returned after successful login.
// This data is extracted from the JWT claims and provided for client convenience.
type UserInfo struct {
	ID       string `json:"id"`       // Unique user identifier
	Username string `json:"username"` // User's login name
	Role     string `json:"role"`     // User's role (admin, uploader, etc.)
}

// Login handles POST /auth/login requests.
// Authenticates users and returns a JWT token for subsequent API requests.
//
// Security Features:
// 1. Rate limiting: Blocks accounts after 5 failed attempts within 15 minutes
// 2. Password hashing: Passwords are bcrypt-hashed, never stored in plaintext
// 3. JWT tokens: Stateless authentication with 24-hour expiration
// 4. Audit logging: All login attempts logged via middleware
//
// Request Flow:
// 1. Parse and validate request body
// 2. Check rate limiter (block if too many failures)
// 3. Authenticate credentials (username + password)
// 4. Record failure/success with rate limiter
// 5. Generate JWT token
// 6. Return token with user info
//
// HTTP Status Codes:
//   - 200 OK: Successful authentication
//   - 400 Bad Request: Invalid request body
//   - 401 Unauthorized: Invalid credentials
//   - 429 Too Many Requests: Rate limit exceeded
//   - 500 Internal Server Error: Server-side error
func (h *AuthHandler) Login(c *gin.Context) {
	// Step 1: Parse and validate request body
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Invalid request body",
			"details": err.Error(),
		}})
		return
	}

	// Step 2: Check rate limiter before attempting authentication
	// This prevents brute force attacks by blocking accounts with too many failures
	allowed, retryAfter := h.rateLimiter.Allow(req.Username)
	if !allowed {
		// Account is blocked - return 429 with retry-after information
		c.Header("Retry-After", retryAfter.String())
		c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{
			"code":        "RATE_LIMIT_EXCEEDED",
			"message":     "Too many failed login attempts. Please try again later.",
			"retry_after": retryAfter.Seconds(), // Seconds until unblocked
		}})
		return
	}

	// Step 3: Attempt authentication with provided credentials
	// The auth service verifies the password hash and generates a JWT token
	token, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		// Authentication failed - record failure and return appropriate error
		h.rateLimiter.RecordFailure(req.Username)

		if err == auth.ErrInvalidCredentials {
			// Invalid username or password - return generic message for security
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "AUTHENTICATION_FAILED",
				"message": "Invalid username or password",
			}})
			return
		}

		// Unexpected error during authentication
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "An error occurred during authentication",
		}})
		return
	}

	// Step 4: Authentication succeeded - clear rate limit counter
	// This resets the failure count so the user can continue normally
	h.rateLimiter.RecordSuccess(req.Username)

	// Step 5: Validate the generated token and extract claims
	// This ensures the token is properly formatted and contains expected data
	claims, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		// Token validation failed (should never happen with freshly generated token)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to validate generated token",
		}})
		return
	}

	// Step 6: Return successful response with token and user information
	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		User:      UserInfo{ID: claims.UserID, Username: claims.Username, Role: claims.Role},
		ExpiresAt: claims.ExpiresAt.Time,
	})
}
