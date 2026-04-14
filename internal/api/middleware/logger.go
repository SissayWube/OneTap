package middleware

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel represents the log level
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         LogLevel               `json:"level"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Method        string                 `json:"method,omitempty"`
	Path          string                 `json:"path,omitempty"`
	Status        int                    `json:"status,omitempty"`
	Duration      float64                `json:"duration_ms,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	Username      string                 `json:"username,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// Logger creates a structured logging middleware
func Logger(level LogLevel) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get correlation ID
		correlationID, _ := c.Get("correlation_id")
		correlationIDStr, _ := correlationID.(string)

		// Get user info if available
		userID, _ := c.Get("user_id")
		userIDStr, _ := userID.(string)
		username, _ := c.Get("username")
		usernameStr, _ := username.(string)

		// Determine log level based on status code
		logLevel := LogLevelInfo
		if c.Writer.Status() >= 500 {
			logLevel = LogLevelError
		} else if c.Writer.Status() >= 400 {
			logLevel = LogLevelWarn
		}

		// Create log entry
		entry := LogEntry{
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Level:         logLevel,
			Message:       "API Request",
			CorrelationID: correlationIDStr,
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Status:        c.Writer.Status(),
			Duration:      float64(duration.Milliseconds()),
			UserID:        userIDStr,
			Username:      usernameStr,
		}

		// Add error if present
		if len(c.Errors) > 0 {
			entry.Error = c.Errors.String()
		}

		// Write log entry as JSON
		logJSON, _ := json.Marshal(entry)
		os.Stdout.Write(logJSON)
		os.Stdout.Write([]byte("\n"))
	}
}

// LogError logs an error with structured format
func LogError(correlationID, errorType, message, stackTrace string) {
	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Level:         LogLevelError,
		Message:       message,
		CorrelationID: correlationID,
		Extra: map[string]interface{}{
			"error_type":  errorType,
			"stack_trace": stackTrace,
		},
	}

	logJSON, _ := json.Marshal(entry)
	os.Stdout.Write(logJSON)
	os.Stdout.Write([]byte("\n"))
}

// LogAuthAttempt logs authentication attempts
func LogAuthAttempt(username string, success bool, correlationID string) {
	level := LogLevelInfo
	message := "Authentication successful"
	if !success {
		level = LogLevelWarn
		message = "Authentication failed"
	}

	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Level:         level,
		Message:       message,
		CorrelationID: correlationID,
		Username:      username,
	}

	logJSON, _ := json.Marshal(entry)
	os.Stdout.Write(logJSON)
	os.Stdout.Write([]byte("\n"))
}

// LogAuthorizationFailure logs authorization failures
func LogAuthorizationFailure(username, resource, requiredRole, correlationID string) {
	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Level:         LogLevelWarn,
		Message:       "Authorization failed",
		CorrelationID: correlationID,
		Username:      username,
		Extra: map[string]interface{}{
			"resource":      resource,
			"required_role": requiredRole,
		},
	}

	logJSON, _ := json.Marshal(entry)
	os.Stdout.Write(logJSON)
	os.Stdout.Write([]byte("\n"))
}
