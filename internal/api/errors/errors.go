package errors

import "net/http"

// AppError defines the interface for application errors
type AppError interface {
	error
	Code() string
	HTTPStatus() int
	Details() interface{}
}

// BaseError implements the AppError interface
type BaseError struct {
	ErrorCode    string      `json:"code"`
	ErrorMessage string      `json:"message"`
	ErrorDetails interface{} `json:"details,omitempty"`
	StatusCode   int         `json:"-"`
}

func (e *BaseError) Error() string {
	return e.ErrorMessage
}

func (e *BaseError) Code() string {
	return e.ErrorCode
}

func (e *BaseError) HTTPStatus() int {
	return e.StatusCode
}

func (e *BaseError) Details() interface{} {
	return e.ErrorDetails
}

// AuthenticationError represents authentication failures
type AuthenticationError struct {
	BaseError
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string, details interface{}) *AuthenticationError {
	return &AuthenticationError{
		BaseError: BaseError{
			ErrorCode:    "AUTHENTICATION_FAILED",
			ErrorMessage: message,
			ErrorDetails: details,
			StatusCode:   http.StatusUnauthorized,
		},
	}
}

// AuthorizationError represents authorization failures
type AuthorizationError struct {
	BaseError
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message string, details interface{}) *AuthorizationError {
	return &AuthorizationError{
		BaseError: BaseError{
			ErrorCode:    "INSUFFICIENT_PERMISSIONS",
			ErrorMessage: message,
			ErrorDetails: details,
			StatusCode:   http.StatusForbidden,
		},
	}
}

// ValidationError represents validation failures
type ValidationError struct {
	BaseError
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details interface{}) *ValidationError {
	return &ValidationError{
		BaseError: BaseError{
			ErrorCode:    "VALIDATION_ERROR",
			ErrorMessage: message,
			ErrorDetails: details,
			StatusCode:   http.StatusBadRequest,
		},
	}
}

// RateLimitError represents rate limit exceeded
type RateLimitError struct {
	BaseError
	RetryAfter int `json:"retry_after"`
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string, retryAfter int) *RateLimitError {
	return &RateLimitError{
		BaseError: BaseError{
			ErrorCode:    "RATE_LIMIT_EXCEEDED",
			ErrorMessage: message,
			ErrorDetails: nil,
			StatusCode:   http.StatusTooManyRequests,
		},
		RetryAfter: retryAfter,
	}
}

// NotFoundError represents resource not found
type NotFoundError struct {
	BaseError
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, details interface{}) *NotFoundError {
	return &NotFoundError{
		BaseError: BaseError{
			ErrorCode:    "RESOURCE_NOT_FOUND",
			ErrorMessage: message,
			ErrorDetails: details,
			StatusCode:   http.StatusNotFound,
		},
	}
}

// InternalError represents internal server errors
type InternalError struct {
	BaseError
}

// NewInternalError creates a new internal error
func NewInternalError(message string, details interface{}) *InternalError {
	return &InternalError{
		BaseError: BaseError{
			ErrorCode:    "INTERNAL_ERROR",
			ErrorMessage: message,
			ErrorDetails: details,
			StatusCode:   http.StatusInternalServerError,
		},
	}
}
