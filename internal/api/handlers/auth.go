package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/onetap/salary-advance-loan-service/internal/auth"
	"github.com/onetap/salary-advance-loan-service/internal/ratelimit"
)

type AuthHandler struct {
	authService auth.AuthService
	rateLimiter ratelimit.RateLimiter
}

func NewAuthHandler(authService auth.AuthService, rateLimiter ratelimit.RateLimiter) *AuthHandler {
	return &AuthHandler{authService: authService, rateLimiter: rateLimiter}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	User      UserInfo  `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Invalid request body",
			"details": err.Error(),
		}})
		return
	}

	allowed, retryAfter := h.rateLimiter.Allow(req.Username)
	if !allowed {
		c.Header("Retry-After", retryAfter.String())
		c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{
			"code":        "RATE_LIMIT_EXCEEDED",
			"message":     "Too many failed login attempts. Please try again later.",
			"retry_after": retryAfter.Seconds(),
		}})
		return
	}

	token, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.rateLimiter.RecordFailure(req.Username)
		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
				"code":    "AUTHENTICATION_FAILED",
				"message": "Invalid username or password",
			}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "An error occurred during authentication",
		}})
		return
	}

	h.rateLimiter.RecordSuccess(req.Username)

	claims, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to validate generated token",
		}})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		User:      UserInfo{ID: claims.UserID, Username: claims.Username, Role: claims.Role},
		ExpiresAt: claims.ExpiresAt.Time,
	})
}
