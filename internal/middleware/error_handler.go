package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "VALIDATION_ERROR"
	ErrorTypeAuthorization ErrorType = "AUTHORIZATION_ERROR"
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeRateLimit     ErrorType = "RATE_LIMIT_EXCEEDED"
	ErrorTypeInternal      ErrorType = "INTERNAL_ERROR"
	ErrorTypePanic         ErrorType = "PANIC_RECOVERED"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType              `json:"type"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
	Cause      error                  `json:"-"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// ErrorResponse represents the JSON error response structure
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Type      ErrorType              `json:"type"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// DefaultErrorHandler implements ErrorHandler interface
type DefaultErrorHandler struct {
	logger            Logger
	includeStackTrace bool
	maskInternalErrs  bool
}

// ErrorHandlerConfig configures the error handler
type ErrorHandlerConfig struct {
	Logger            Logger
	IncludeStackTrace bool // Include stack trace in error responses (for development)
	MaskInternalErrs  bool // Mask internal error details (for production)
}

// NewDefaultErrorHandler creates a new default error handler
func NewDefaultErrorHandler(config ErrorHandlerConfig) ErrorHandler {
	return &DefaultErrorHandler{
		logger:            config.Logger,
		includeStackTrace: config.IncludeStackTrace,
		maskInternalErrs:  config.MaskInternalErrs,
	}
}

// HandleError implements the ErrorHandler interface
func (h *DefaultErrorHandler) HandleError(ctx HTTPContext, err error) error {
	if err == nil {
		return nil
	}

	req := ctx.Request()
	resp := ctx.Response()

	// Extract request ID from context
	requestID := ""
	if id, exists := ctx.Get("requestID"); exists {
		if idStr, ok := id.(string); ok {
			requestID = idStr
		}
	}

	// Convert error to AppError
	appErr := h.convertToAppError(err, requestID)

	// Log the error
	if h.logger != nil {
		logCtx := context.Background()
		if requestID != "" {
			logCtx = context.WithValue(logCtx, "request_id", requestID)
		}

		fields := []Field{
			{Key: "error_type", Value: string(appErr.Type)},
			{Key: "error_code", Value: appErr.Code},
			{Key: "path", Value: req.Path()},
			{Key: "method", Value: req.Method()},
			{Key: "ip", Value: req.IP()},
			{Key: "request_id", Value: requestID},
		}

		if appErr.StatusCode >= 500 {
			h.logger.Error(logCtx, "Internal server error", appErr.Cause, fields...)
		} else {
			h.logger.Warn(logCtx, "Client error", fields...)
		}
	}

	// Create error response
	errorResp := &ErrorResponse{
		Error:     appErr.Message,
		Type:      appErr.Type,
		Code:      appErr.Code,
		Message:   appErr.Message,
		Details:   appErr.Details,
		Fields:    appErr.Fields,
		Path:      req.Path(),
		Method:    req.Method(),
		Timestamp: appErr.Timestamp,
		RequestID: requestID,
	}

	// Mask internal error details in production
	if h.maskInternalErrs && appErr.StatusCode >= 500 {
		errorResp.Error = "Internal server error"
		errorResp.Message = "An unexpected error occurred"
		errorResp.Details = ""
		errorResp.Fields = nil
	}

	// Set response status and return JSON
	resp.SetStatus(appErr.StatusCode)
	return resp.JSON(errorResp)
}

// convertToAppError converts various error types to AppError
func (h *DefaultErrorHandler) convertToAppError(err error, requestID string) *AppError {
	now := time.Now()

	// If it's already an AppError, return it with request ID
	var appErr *AppError
	if errors.As(err, &appErr) {
		appErr.RequestID = requestID
		if appErr.Timestamp.IsZero() {
			appErr.Timestamp = now
		}
		return appErr
	}

	// Check for specific error types
	switch {
	case isValidationError(err):
		return &AppError{
			Type:       ErrorTypeValidation,
			Code:       "VALIDATION_FAILED",
			Message:    "Request validation failed",
			Details:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Cause:      err,
			Timestamp:  now,
			RequestID:  requestID,
		}

	case isNotFoundError(err):
		return &AppError{
			Type:       ErrorTypeNotFound,
			Code:       "RESOURCE_NOT_FOUND",
			Message:    "The requested resource was not found",
			Details:    err.Error(),
			StatusCode: http.StatusNotFound,
			Cause:      err,
			Timestamp:  now,
			RequestID:  requestID,
		}

	case isAuthorizationError(err):
		return &AppError{
			Type:       ErrorTypeAuthorization,
			Code:       "AUTHORIZATION_FAILED",
			Message:    "Access denied",
			Details:    err.Error(),
			StatusCode: http.StatusForbidden,
			Cause:      err,
			Timestamp:  now,
			RequestID:  requestID,
		}

	default:
		return &AppError{
			Type:       ErrorTypeInternal,
			Code:       "INTERNAL_ERROR",
			Message:    "Internal server error",
			Details:    err.Error(),
			StatusCode: http.StatusInternalServerError,
			Cause:      err,
			Timestamp:  now,
			RequestID:  requestID,
		}
	}
}

// Helper functions to identify error types
func isValidationError(err error) bool {
	// Add logic to identify validation errors
	// This could check for specific error types or error message patterns
	return false
}

func isNotFoundError(err error) bool {
	// Add logic to identify not found errors
	return false
}

func isAuthorizationError(err error) bool {
	// Add logic to identify authorization errors
	return false
}

// RecoveryMiddleware handles panic recovery
type RecoveryMiddleware struct {
	errorHandler      ErrorHandler
	logger            Logger
	includeStackTrace bool
}

// RecoveryConfig configures the recovery middleware
type RecoveryConfig struct {
	ErrorHandler      ErrorHandler
	Logger            Logger
	IncludeStackTrace bool
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(config RecoveryConfig) Middleware {
	return &RecoveryMiddleware{
		errorHandler:      config.ErrorHandler,
		logger:            config.Logger,
		includeStackTrace: config.IncludeStackTrace,
	}
}

// Handle implements the Middleware interface
func (m *RecoveryMiddleware) Handle(ctx HTTPContext) error {
	defer func() {
		if r := recover(); r != nil {
			// Extract request information
			req := ctx.Request()
			requestID := ""
			if id, exists := ctx.Get("requestID"); exists {
				if idStr, ok := id.(string); ok {
					requestID = idStr
				}
			}

			// Create panic error
			var err error
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = fmt.Errorf("panic: %v", v)
			}

			// Get stack trace
			stackTrace := ""
			if m.includeStackTrace {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stackTrace = string(buf[:n])
			}

			// Log the panic
			if m.logger != nil {
				logCtx := context.Background()
				if requestID != "" {
					logCtx = context.WithValue(logCtx, "request_id", requestID)
				}

				fields := []Field{
					{Key: "path", Value: req.Path()},
					{Key: "method", Value: req.Method()},
					{Key: "ip", Value: req.IP()},
					{Key: "request_id", Value: requestID},
				}

				if stackTrace != "" {
					fields = append(fields, Field{Key: "stack_trace", Value: stackTrace})
				}

				m.logger.Error(logCtx, "Panic recovered", err, fields...)
			}

			// Create panic error
			panicErr := &AppError{
				Type:       ErrorTypePanic,
				Code:       "PANIC_RECOVERED",
				Message:    "An unexpected error occurred",
				Details:    err.Error(),
				StatusCode: http.StatusInternalServerError,
				Cause:      err,
				Timestamp:  time.Now(),
				RequestID:  requestID,
			}

			if stackTrace != "" {
				panicErr.Fields = map[string]interface{}{
					"stack_trace": stackTrace,
				}
			}

			// Handle the error
			if m.errorHandler != nil {
				_ = m.errorHandler.HandleError(ctx, panicErr)
			} else {
				// Fallback error handling
				ctx.Response().SetStatus(http.StatusInternalServerError)
				_ = ctx.Response().JSON(map[string]interface{}{
					"error":      "Internal server error",
					"message":    "An unexpected error occurred",
					"request_id": requestID,
				})
			}
		}
	}()

	return ctx.Next()
}

// ErrorMiddleware wraps an error handler as middleware
type ErrorMiddleware struct {
	errorHandler ErrorHandler
}

// NewErrorMiddleware creates a new error middleware
func NewErrorMiddleware(errorHandler ErrorHandler) Middleware {
	return &ErrorMiddleware{
		errorHandler: errorHandler,
	}
}

// Handle implements the Middleware interface
func (m *ErrorMiddleware) Handle(ctx HTTPContext) error {
	err := ctx.Next()
	if err != nil {
		return m.errorHandler.HandleError(ctx, err)
	}
	return nil
}