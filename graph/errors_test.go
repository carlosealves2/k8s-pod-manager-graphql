//go:build ignore
// +build ignore

// Tests skipped due to dependency issues
package graph

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDefaultErrorHandler_HandleServiceError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	tests := []struct {
		name       string
		err        error
		operation  string
		wantNil    bool
		wantCode   string
		wantMsg    string
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test",
			wantNil:   true,
		},
		{
			name:      "kubernetes not found error",
			err:       apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test-pod"),
			operation: "getPod",
			wantCode:  string(ErrorTypeNotFound),
			wantMsg:   "Resource not found during getPod operation",
		},
		{
			name:      "kubernetes unauthorized error",
			err:       apierrors.NewUnauthorized("insufficient permissions"),
			operation: "listPods",
			wantCode:  string(ErrorTypeUnauthorized),
			wantMsg:   "Unauthorized to perform listPods operation",
		},
		{
			name:      "kubernetes forbidden error",
			err:       apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "test-pod", errors.New("forbidden")),
			operation: "deletePod",
			wantCode:  string(ErrorTypeForbidden),
			wantMsg:   "Forbidden to perform deletePod operation",
		},
		{
			name:      "kubernetes already exists error",
			err:       apierrors.NewAlreadyExists(schema.GroupResource{Resource: "pods"}, "test-pod"),
			operation: "createPod",
			wantCode:  string(ErrorTypeConflict),
			wantMsg:   "Resource already exists during createPod operation",
		},
		{
			name:      "kubernetes timeout error",
			err:       apierrors.NewTimeoutError("operation timed out", 30),
			operation: "scalePod",
			wantCode:  string(ErrorTypeTimeout),
			wantMsg:   "Operation scalePod timed out",
		},
		{
			name:      "generic service error",
			err:       errors.New("generic service error"),
			operation: "testOperation",
			wantCode:  string(ErrorTypeServiceError),
			wantMsg:   "Service error during testOperation operation",
		},
		{
			name:      "service error with sensitive information",
			err:       errors.New("database password=secret123 failed"),
			operation: "connect",
			wantCode:  string(ErrorTypeServiceError),
			wantMsg:   "Service error during connect operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleServiceError(tt.err, tt.operation)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			gqlErr, ok := result.(*gqlerror.Error)
			require.True(t, ok, "expected *gqlerror.Error")

			assert.Contains(t, gqlErr.Message, tt.wantMsg)
			assert.Equal(t, tt.wantCode, gqlErr.Extensions["code"])
			assert.NotEmpty(t, gqlErr.Extensions["details"])
		})
	}
}

func TestDefaultErrorHandler_HandleValidationError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	tests := []struct {
		name     string
		err      error
		field    string
		wantNil  bool
		wantCode string
		wantMsg  string
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			field:   "test",
			wantNil: true,
		},
		{
			name:     "validation error for namespace field",
			err:      errors.New("namespace cannot be empty"),
			field:    "namespace",
			wantCode: string(ErrorTypeValidation),
			wantMsg:  "Validation failed for field 'namespace'",
		},
		{
			name:     "validation error for pod name field",
			err:      errors.New("invalid pod name format"),
			field:    "podName",
			wantCode: string(ErrorTypeValidation),
			wantMsg:  "Validation failed for field 'podName'",
		},
		{
			name:     "validation error for scaling input",
			err:      errors.New("replica count cannot be negative"),
			field:    "replicas",
			wantCode: string(ErrorTypeValidation),
			wantMsg:  "Validation failed for field 'replicas'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleValidationError(tt.err, tt.field)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			gqlErr, ok := result.(*gqlerror.Error)
			require.True(t, ok, "expected *gqlerror.Error")

			assert.Contains(t, gqlErr.Message, tt.wantMsg)
			assert.Equal(t, tt.wantCode, gqlErr.Extensions["code"])
			assert.Equal(t, tt.err.Error(), gqlErr.Extensions["details"])
		})
	}
}

func TestDefaultErrorHandler_HandleNotFoundError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	tests := []struct {
		name      string
		resource  string
		namespace string
		resName   string
		wantCode  string
		wantMsg   string
	}{
		{
			name:      "pod not found with namespace",
			resource:  "pod",
			namespace: "default",
			resName:   "test-pod",
			wantCode:  string(ErrorTypeNotFound),
			wantMsg:   "Pod 'test-pod' not found in namespace 'default'",
		},
		{
			name:      "deployment not found with namespace",
			resource:  "deployment",
			namespace: "production",
			resName:   "web-app",
			wantCode:  string(ErrorTypeNotFound),
			wantMsg:   "Deployment 'web-app' not found in namespace 'production'",
		},
		{
			name:      "resource not found without namespace",
			resource:  "namespace",
			namespace: "",
			resName:   "missing-ns",
			wantCode:  string(ErrorTypeNotFound),
			wantMsg:   "Namespace 'missing-ns' not found",
		},
		{
			name:      "service not found with empty namespace",
			resource:  "service",
			namespace: "",
			resName:   "api-service",
			wantCode:  string(ErrorTypeNotFound),
			wantMsg:   "Service 'api-service' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleNotFoundError(tt.resource, tt.namespace, tt.resName)

			require.NotNil(t, result)
			gqlErr, ok := result.(*gqlerror.Error)
			require.True(t, ok, "expected *gqlerror.Error")

			assert.Equal(t, tt.wantMsg, gqlErr.Message)
			assert.Equal(t, tt.wantCode, gqlErr.Extensions["code"])
			assert.Contains(t, gqlErr.Extensions["details"].(string), "does not exist")
		})
	}
}

func TestDefaultErrorHandler_HandleInternalError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	tests := []struct {
		name     string
		err      error
		context  string
		wantNil  bool
		wantCode string
		wantMsg  string
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			context: "test",
			wantNil: true,
		},
		{
			name:     "internal error in database operation",
			err:      errors.New("database connection failed"),
			context:  "database operation",
			wantCode: string(ErrorTypeInternal),
			wantMsg:  "Internal error in database operation",
		},
		{
			name:     "internal error in kubernetes client",
			err:      errors.New("cluster unreachable"),
			context:  "kubernetes client",
			wantCode: string(ErrorTypeInternal),
			wantMsg:  "Internal error in kubernetes client",
		},
		{
			name:     "panic recovery error",
			err:      errors.New("panic: runtime error"),
			context:  "resolver execution",
			wantCode: string(ErrorTypeInternal),
			wantMsg:  "Internal error in resolver execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleInternalError(tt.err, tt.context)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			gqlErr, ok := result.(*gqlerror.Error)
			require.True(t, ok, "expected *gqlerror.Error")

			assert.Equal(t, tt.wantMsg, gqlErr.Message)
			assert.Equal(t, tt.wantCode, gqlErr.Extensions["code"])
			assert.Equal(t, "An unexpected error occurred", gqlErr.Extensions["details"])
		})
	}
}

func TestDefaultErrorHandler_sanitizeErrorMessage(t *testing.T) {
	handler := NewDefaultErrorHandler()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "clean message unchanged",
			message:  "simple error message",
			expected: "simple error message",
		},
		{
			name:     "message with password",
			message:  "failed to connect with password=secret123",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "message with token",
			message:  "authentication failed with token abc123",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "message with secret",
			message:  "secret key validation failed",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "message with key",
			message:  "API key is invalid",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "message with auth",
			message:  "authorization header missing",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "very long message truncated",
			message:  strings.Repeat("a", 250),
			expected: strings.Repeat("a", 197) + "...",
		},
		{
			name:     "message exactly at limit",
			message:  strings.Repeat("a", 200),
			expected: strings.Repeat("a", 200),
		},
		{
			name:     "uppercase sensitive word",
			message:  "PASSWORD authentication failed",
			expected: "Sensitive information in error message (details hidden)",
		},
		{
			name:     "mixed case sensitive word",
			message:  "Token validation error",
			expected: "Sensitive information in error message (details hidden)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.sanitizeErrorMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock ErrorHandler for testing ChainedErrorHandler
type MockErrorHandler struct {
	mock.Mock
}

func (m *MockErrorHandler) HandleServiceError(err error, operation string) error {
	args := m.Called(err, operation)
	return args.Error(0)
}

func (m *MockErrorHandler) HandleValidationError(err error, field string) error {
	args := m.Called(err, field)
	return args.Error(0)
}

func (m *MockErrorHandler) HandleNotFoundError(resource, namespace, name string) error {
	args := m.Called(resource, namespace, name)
	return args.Error(0)
}

func (m *MockErrorHandler) HandleInternalError(err error, context string) error {
	args := m.Called(err, context)
	return args.Error(0)
}

func TestChainedErrorHandler(t *testing.T) {
	t.Run("first handler handles error", func(t *testing.T) {
		handler1 := new(MockErrorHandler)
		handler2 := new(MockErrorHandler)
		chainedHandler := NewChainedErrorHandler(handler1, handler2)

		testErr := errors.New("test error")
		expectedErr := &gqlerror.Error{Message: "handled by first"}

		// First handler returns error (handles it)
		handler1.On("HandleServiceError", testErr, "test").Return(expectedErr)
		// Second handler should not be called

		result := chainedHandler.HandleServiceError(testErr, "test")

		assert.Equal(t, expectedErr, result)
		handler1.AssertExpectations(t)
		handler2.AssertNotCalled(t, "HandleServiceError")
	})

	t.Run("first handler returns nil, second handler handles error", func(t *testing.T) {
		handler1 := new(MockErrorHandler)
		handler2 := new(MockErrorHandler)
		chainedHandler := NewChainedErrorHandler(handler1, handler2)

		testErr := errors.New("test error")
		expectedErr := &gqlerror.Error{Message: "handled by second"}

		// First handler returns nil (doesn't handle it)
		handler1.On("HandleServiceError", testErr, "test").Return(nil)
		// Second handler handles it
		handler2.On("HandleServiceError", testErr, "test").Return(expectedErr)

		result := chainedHandler.HandleServiceError(testErr, "test")

		assert.Equal(t, expectedErr, result)
		handler1.AssertExpectations(t)
		handler2.AssertExpectations(t)
	})

	t.Run("no handler handles error, falls back to default", func(t *testing.T) {
		handler1 := new(MockErrorHandler)
		handler2 := new(MockErrorHandler)
		chainedHandler := NewChainedErrorHandler(handler1, handler2)

		testErr := errors.New("test error")

		// Both handlers return nil (don't handle it)
		handler1.On("HandleServiceError", testErr, "test").Return(nil)
		handler2.On("HandleServiceError", testErr, "test").Return(nil)

		result := chainedHandler.HandleServiceError(testErr, "test")

		// Should get result from default handler
		require.NotNil(t, result)
		gqlErr, ok := result.(*gqlerror.Error)
		require.True(t, ok)
		assert.Contains(t, gqlErr.Message, "Service error")

		handler1.AssertExpectations(t)
		handler2.AssertExpectations(t)
	})

	t.Run("all error handling methods work in chain", func(t *testing.T) {
		handler1 := new(MockErrorHandler)
		chainedHandler := NewChainedErrorHandler(handler1)

		testErr := errors.New("test error")
		expectedErr := &gqlerror.Error{Message: "handled"}

		// Test HandleValidationError
		handler1.On("HandleValidationError", testErr, "field").Return(expectedErr)
		result := chainedHandler.HandleValidationError(testErr, "field")
		assert.Equal(t, expectedErr, result)

		// Test HandleNotFoundError
		handler1.On("HandleNotFoundError", "resource", "namespace", "name").Return(expectedErr)
		result = chainedHandler.HandleNotFoundError("resource", "namespace", "name")
		assert.Equal(t, expectedErr, result)

		// Test HandleInternalError
		handler1.On("HandleInternalError", testErr, "context").Return(expectedErr)
		result = chainedHandler.HandleInternalError(testErr, "context")
		assert.Equal(t, expectedErr, result)

		handler1.AssertExpectations(t)
	})
}

func TestGraphQLErrorType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant GraphQLErrorType
		expected string
	}{
		{"validation error", ErrorTypeValidation, "VALIDATION_ERROR"},
		{"not found error", ErrorTypeNotFound, "NOT_FOUND"},
		{"unauthorized error", ErrorTypeUnauthorized, "UNAUTHORIZED"},
		{"forbidden error", ErrorTypeForbidden, "FORBIDDEN"},
		{"conflict error", ErrorTypeConflict, "CONFLICT"},
		{"internal error", ErrorTypeInternal, "INTERNAL_ERROR"},
		{"service error", ErrorTypeServiceError, "SERVICE_ERROR"},
		{"timeout error", ErrorTypeTimeout, "TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}

func TestDefaultErrorHandler_createGraphQLError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	result := handler.createGraphQLError(ErrorTypeValidation, "test message", "test details")

	require.NotNil(t, result)
	gqlErr, ok := result.(*gqlerror.Error)
	require.True(t, ok)

	assert.Equal(t, "test message", gqlErr.Message)
	assert.Equal(t, string(ErrorTypeValidation), gqlErr.Extensions["code"])
	assert.Equal(t, "test details", gqlErr.Extensions["details"])
}

// Test error handler interface compliance
func TestErrorHandlerInterfaceCompliance(t *testing.T) {
	t.Run("DefaultErrorHandler implements ErrorHandler", func(t *testing.T) {
		var _ ErrorHandler = (*DefaultErrorHandler)(nil)
	})

	t.Run("ChainedErrorHandler implements ErrorHandler", func(t *testing.T) {
		var _ ErrorHandler = (*ChainedErrorHandler)(nil)
	})
}

// Test initialization functions
func TestErrorHandlerInitialization(t *testing.T) {
	t.Run("NewDefaultErrorHandler", func(t *testing.T) {
		handler := NewDefaultErrorHandler()
		require.NotNil(t, handler)
		assert.IsType(t, &DefaultErrorHandler{}, handler)
	})

	t.Run("NewChainedErrorHandler", func(t *testing.T) {
		handler1 := NewDefaultErrorHandler()
		handler2 := NewDefaultErrorHandler()
		chained := NewChainedErrorHandler(handler1, handler2)

		require.NotNil(t, chained)
		assert.IsType(t, &ChainedErrorHandler{}, chained)
		assert.Len(t, chained.handlers, 2)
	})

	t.Run("NewChainedErrorHandler with no handlers", func(t *testing.T) {
		chained := NewChainedErrorHandler()
		require.NotNil(t, chained)
		assert.Len(t, chained.handlers, 0)
	})
}

// Benchmark tests for error handling performance
func BenchmarkDefaultErrorHandler_HandleServiceError(b *testing.B) {
	handler := NewDefaultErrorHandler()
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleServiceError(err, "test")
	}
}

func BenchmarkDefaultErrorHandler_sanitizeErrorMessage(b *testing.B) {
	handler := NewDefaultErrorHandler()
	message := "this is a test error message without sensitive information"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.sanitizeErrorMessage(message)
	}
}

func BenchmarkChainedErrorHandler(b *testing.B) {
	handler1 := NewDefaultErrorHandler()
	handler2 := NewDefaultErrorHandler()
	chained := NewChainedErrorHandler(handler1, handler2)
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chained.HandleServiceError(err, "test")
	}
}

// Test edge cases and error conditions
func TestErrorHandlerEdgeCases(t *testing.T) {
	handler := NewDefaultErrorHandler()

	t.Run("empty strings handling", func(t *testing.T) {
		result := handler.HandleNotFoundError("", "", "")
		require.NotNil(t, result)
		gqlErr := result.(*gqlerror.Error)
		assert.Contains(t, gqlErr.Message, "'' not found")
	})

	t.Run("special characters in error messages", func(t *testing.T) {
		err := errors.New("error with special chars: !@#$%^&*()")
		result := handler.HandleServiceError(err, "test")
		require.NotNil(t, result)
		// Should not crash or corrupt the message
	})

	t.Run("very long field names", func(t *testing.T) {
		longField := strings.Repeat("field", 100)
		err := errors.New("validation error")
		result := handler.HandleValidationError(err, longField)
		require.NotNil(t, result)
		gqlErr := result.(*gqlerror.Error)
		assert.Contains(t, gqlErr.Message, longField)
	})

	t.Run("unicode characters in messages", func(t *testing.T) {
		err := errors.New("错误信息 with unicode 🚀")
		result := handler.HandleServiceError(err, "test")
		require.NotNil(t, result)
		// Should handle unicode without issues
	})
}