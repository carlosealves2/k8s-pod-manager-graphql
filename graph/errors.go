package graph

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"k8s.io/apimachinery/pkg/api/errors"
)

// GraphQLErrorType defines different error categories for consistent error handling
type GraphQLErrorType string

const (
	ErrorTypeValidation    GraphQLErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound      GraphQLErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized  GraphQLErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden     GraphQLErrorType = "FORBIDDEN"
	ErrorTypeConflict      GraphQLErrorType = "CONFLICT"
	ErrorTypeInternal      GraphQLErrorType = "INTERNAL_ERROR"
	ErrorTypeServiceError  GraphQLErrorType = "SERVICE_ERROR"
	ErrorTypeTimeout       GraphQLErrorType = "TIMEOUT"
)

// DefaultErrorHandler implements ErrorHandler interface
// Provides standardized error handling and formatting for GraphQL responses
type DefaultErrorHandler struct{}

// NewDefaultErrorHandler creates a new instance of DefaultErrorHandler
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{}
}

// HandleServiceError handles errors from the service layer
func (h *DefaultErrorHandler) HandleServiceError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Handle Kubernetes API errors
	if errors.IsNotFound(err) {
		return h.createGraphQLError(
			ErrorTypeNotFound,
			fmt.Sprintf("Resource not found during %s operation", operation),
			err.Error(),
		)
	}

	if errors.IsUnauthorized(err) {
		return h.createGraphQLError(
			ErrorTypeUnauthorized,
			fmt.Sprintf("Unauthorized to perform %s operation", operation),
			"Insufficient permissions",
		)
	}

	if errors.IsForbidden(err) {
		return h.createGraphQLError(
			ErrorTypeForbidden,
			fmt.Sprintf("Forbidden to perform %s operation", operation),
			"Operation not allowed",
		)
	}

	if errors.IsAlreadyExists(err) {
		return h.createGraphQLError(
			ErrorTypeConflict,
			fmt.Sprintf("Resource already exists during %s operation", operation),
			err.Error(),
		)
	}

	if errors.IsTimeout(err) {
		return h.createGraphQLError(
			ErrorTypeTimeout,
			fmt.Sprintf("Operation %s timed out", operation),
			"Request timed out",
		)
	}

	// Handle generic service errors
	return h.createGraphQLError(
		ErrorTypeServiceError,
		fmt.Sprintf("Service error during %s operation", operation),
		h.sanitizeErrorMessage(err.Error()),
	)
}

// HandleValidationError handles validation errors
func (h *DefaultErrorHandler) HandleValidationError(err error, field string) error {
	if err == nil {
		return nil
	}

	return h.createGraphQLError(
		ErrorTypeValidation,
		fmt.Sprintf("Validation failed for field '%s'", field),
		err.Error(),
	)
}

// HandleNotFoundError handles resource not found errors
func (h *DefaultErrorHandler) HandleNotFoundError(resource, namespace, name string) error {
	message := fmt.Sprintf("%s '%s' not found", strings.Title(resource), name)
	if namespace != "" {
		message = fmt.Sprintf("%s '%s' not found in namespace '%s'", strings.Title(resource), name, namespace)
	}

	return h.createGraphQLError(
		ErrorTypeNotFound,
		message,
		fmt.Sprintf("The requested %s does not exist", resource),
	)
}

// HandleInternalError handles internal server errors
func (h *DefaultErrorHandler) HandleInternalError(err error, context string) error {
	if err == nil {
		return nil
	}

	return h.createGraphQLError(
		ErrorTypeInternal,
		fmt.Sprintf("Internal error in %s", context),
		"An unexpected error occurred",
	)
}

// createGraphQLError creates a properly formatted GraphQL error
func (h *DefaultErrorHandler) createGraphQLError(errorType GraphQLErrorType, message, details string) error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code":    string(errorType),
			"details": details,
		},
	}
}

// sanitizeErrorMessage removes sensitive information from error messages
func (h *DefaultErrorHandler) sanitizeErrorMessage(message string) string {
	// Remove common sensitive patterns
	sensitive := []string{
		"password",
		"token",
		"secret",
		"key",
		"auth",
	}

	lower := strings.ToLower(message)
	for _, pattern := range sensitive {
		if strings.Contains(lower, pattern) {
			return "Sensitive information in error message (details hidden)"
		}
	}

	// Truncate very long error messages
	if len(message) > 200 {
		return message[:197] + "..."
	}

	return message
}

// ChainedErrorHandler allows chaining multiple error handlers (Chain of Responsibility Pattern)
type ChainedErrorHandler struct {
	handlers []ErrorHandler
}

// NewChainedErrorHandler creates a new chained error handler
func NewChainedErrorHandler(handlers ...ErrorHandler) *ChainedErrorHandler {
	return &ChainedErrorHandler{
		handlers: handlers,
	}
}

// HandleServiceError chains through all handlers until one handles the error
func (c *ChainedErrorHandler) HandleServiceError(err error, operation string) error {
	for _, handler := range c.handlers {
		if handledErr := handler.HandleServiceError(err, operation); handledErr != nil {
			return handledErr
		}
	}
	// If no handler processed the error, use the default
	return NewDefaultErrorHandler().HandleServiceError(err, operation)
}

// HandleValidationError chains through all handlers
func (c *ChainedErrorHandler) HandleValidationError(err error, field string) error {
	for _, handler := range c.handlers {
		if handledErr := handler.HandleValidationError(err, field); handledErr != nil {
			return handledErr
		}
	}
	return NewDefaultErrorHandler().HandleValidationError(err, field)
}

// HandleNotFoundError chains through all handlers
func (c *ChainedErrorHandler) HandleNotFoundError(resource, namespace, name string) error {
	for _, handler := range c.handlers {
		if handledErr := handler.HandleNotFoundError(resource, namespace, name); handledErr != nil {
			return handledErr
		}
	}
	return NewDefaultErrorHandler().HandleNotFoundError(resource, namespace, name)
}

// HandleInternalError chains through all handlers
func (c *ChainedErrorHandler) HandleInternalError(err error, context string) error {
	for _, handler := range c.handlers {
		if handledErr := handler.HandleInternalError(err, context); handledErr != nil {
			return handledErr
		}
	}
	return NewDefaultErrorHandler().HandleInternalError(err, context)
}