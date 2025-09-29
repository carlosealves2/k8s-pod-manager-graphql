package handlers

import (
	"fmt"
	"net/http"
	"time"
)

// StandardResponse provides consistent response structure
type StandardResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorInfo provides structured error information
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Checks    map[string]interface{} `json:"checks,omitempty"`
	Uptime    string                 `json:"uptime"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
}

// InfoResponse represents service information response
type InfoResponse struct {
	Service    *ServiceInfo           `json:"service"`
	Kubernetes *ServerVersionInfo     `json:"kubernetes,omitempty"`
	Database   *DatabaseStats         `json:"database,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ServiceInfo represents service information
type ServiceInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

// ErrorCode represents standard error codes
type ErrorCode string

const (
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeBadRequest     ErrorCode = "BAD_REQUEST"
	ErrCodeServiceUnavail ErrorCode = "SERVICE_UNAVAILABLE"
)

// ResponseBuilder helps build standardized responses
type ResponseBuilder struct {
	requestID string
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(requestID string) *ResponseBuilder {
	return &ResponseBuilder{requestID: requestID}
}

// Success builds a success response
func (rb *ResponseBuilder) Success(data interface{}) *StandardResponse {
	return &StandardResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: rb.requestID,
	}
}

// Error builds an error response
func (rb *ResponseBuilder) Error(code ErrorCode, message string, details interface{}) *StandardResponse {
	return &StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(code),
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: rb.requestID,
	}
}

// ValidationError builds a validation error response
func (rb *ResponseBuilder) ValidationError(message string, details interface{}) *StandardResponse {
	return rb.Error(ErrCodeValidation, message, details)
}

// NotFoundError builds a not found error response
func (rb *ResponseBuilder) NotFoundError(resource string) *StandardResponse {
	return rb.Error(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

// InternalError builds an internal error response
func (rb *ResponseBuilder) InternalError(message string) *StandardResponse {
	return rb.Error(ErrCodeInternal, message, nil)
}

// HTTPStatusFromErrorCode maps error codes to HTTP status codes
func HTTPStatusFromErrorCode(code ErrorCode) int {
	switch code {
	case ErrCodeValidation, ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}