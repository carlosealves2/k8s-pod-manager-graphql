package services

import (
	"errors"
	"fmt"
)

// ServiceError represents different types of service errors
type ServiceError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %s)", e.Type, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// ErrorType defines different categories of service errors
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeUnavailable  ErrorType = "SERVICE_UNAVAILABLE"
)

// Error constructors for common scenarios
func NewValidationError(message string, cause error) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   cause,
	}
}

func NewNotFoundError(resource, identifier string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeNotFound,
		Message: fmt.Sprintf("%s '%s' not found", resource, identifier),
	}
}

func NewInternalError(message string, cause error) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   cause,
	}
}

func NewConflictError(message string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeConflict,
		Message: message,
	}
}

func NewUnavailableError(message string, cause error) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeUnavailable,
		Message: message,
		Cause:   cause,
	}
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Type == errorType
	}
	return false
}

// PodOperationError represents pod operation specific errors
type PodOperationError struct {
	Operation string
	PodName   string
	Namespace string
	Reason    string
	Cause     error
}

func (e *PodOperationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("pod operation '%s' failed for pod %s/%s: %s (caused by: %s)",
			e.Operation, e.Namespace, e.PodName, e.Reason, e.Cause.Error())
	}
	return fmt.Sprintf("pod operation '%s' failed for pod %s/%s: %s",
		e.Operation, e.Namespace, e.PodName, e.Reason)
}

func (e *PodOperationError) Unwrap() error {
	return e.Cause
}

func NewPodOperationError(operation, podName, namespace, reason string, cause error) *PodOperationError {
	return &PodOperationError{
		Operation: operation,
		PodName:   podName,
		Namespace: namespace,
		Reason:    reason,
		Cause:     cause,
	}
}