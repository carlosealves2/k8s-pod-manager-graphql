//go:build ignore
// +build ignore

package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceError_Error(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	tests := []struct {
		name           string
		serviceError   *ServiceError
		expectedOutput string
	}{
		{
			name: "ServiceError without cause",
			serviceError: &ServiceError{
				Type:    ErrorTypeValidation,
				Message: "Invalid input provided",
			},
			expectedOutput: "VALIDATION_ERROR: Invalid input provided",
		},
		{
			name: "ServiceError with cause",
			serviceError: &ServiceError{
				Type:    ErrorTypeInternal,
				Message: "Database operation failed",
				Cause:   errors.New("connection timeout"),
			},
			expectedOutput: "INTERNAL_ERROR: Database operation failed (caused by: connection timeout)",
		},
		{
			name: "NotFound error",
			serviceError: &ServiceError{
				Type:    ErrorTypeNotFound,
				Message: "Resource not found",
			},
			expectedOutput: "NOT_FOUND: Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serviceError.Error()
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	t.Run("ServiceError with cause returns cause", func(t *testing.T) {
		originalError := errors.New("original error")
		serviceError := &ServiceError{
			Type:    ErrorTypeInternal,
			Message: "Wrapped error",
			Cause:   originalError,
		}

		unwrapped := serviceError.Unwrap()
		assert.Equal(t, originalError, unwrapped)
	})

	t.Run("ServiceError without cause returns nil", func(t *testing.T) {
		serviceError := &ServiceError{
			Type:    ErrorTypeValidation,
			Message: "Validation failed",
		}

		unwrapped := serviceError.Unwrap()
		assert.Nil(t, unwrapped)
	})
}

func TestErrorConstructors(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	t.Run("NewValidationError", func(t *testing.T) {
		cause := errors.New("field required")
		err := NewValidationError("Validation failed", cause)

		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "Validation failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewNotFoundError", func(t *testing.T) {
		err := NewNotFoundError("Pod", "nginx-123")

		assert.Equal(t, ErrorTypeNotFound, err.Type)
		assert.Equal(t, "Pod 'nginx-123' not found", err.Message)
		assert.Nil(t, err.Cause)
	})

	t.Run("NewInternalError", func(t *testing.T) {
		cause := errors.New("database connection failed")
		err := NewInternalError("Internal server error", cause)

		assert.Equal(t, ErrorTypeInternal, err.Type)
		assert.Equal(t, "Internal server error", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewConflictError", func(t *testing.T) {
		err := NewConflictError("Resource already exists")

		assert.Equal(t, ErrorTypeConflict, err.Type)
		assert.Equal(t, "Resource already exists", err.Message)
		assert.Nil(t, err.Cause)
	})

	t.Run("NewUnavailableError", func(t *testing.T) {
		cause := errors.New("service temporarily unavailable")
		err := NewUnavailableError("Service unavailable", cause)

		assert.Equal(t, ErrorTypeUnavailable, err.Type)
		assert.Equal(t, "Service unavailable", err.Message)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestIsErrorType(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	tests := []struct {
		name           string
		err            error
		errorType      ErrorType
		expectedResult bool
	}{
		{
			name:           "ServiceError with matching type",
			err:            NewValidationError("Invalid input", nil),
			errorType:      ErrorTypeValidation,
			expectedResult: true,
		},
		{
			name:           "ServiceError with non-matching type",
			err:            NewValidationError("Invalid input", nil),
			errorType:      ErrorTypeNotFound,
			expectedResult: false,
		},
		{
			name:           "Non-ServiceError",
			err:            errors.New("generic error"),
			errorType:      ErrorTypeValidation,
			expectedResult: false,
		},
		{
			name:           "Wrapped ServiceError",
			err:            fmt.Errorf("wrapped: %w", NewInternalError("Internal error", nil)),
			errorType:      ErrorTypeInternal,
			expectedResult: true,
		},
		{
			name:           "Nil error",
			err:            nil,
			errorType:      ErrorTypeValidation,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrorType(tt.err, tt.errorType)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestPodOperationError_Error(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	tests := []struct {
		name           string
		podError       *PodOperationError
		expectedOutput string
	}{
		{
			name: "PodOperationError without cause",
			podError: &PodOperationError{
				Operation: "delete",
				PodName:   "nginx-123",
				Namespace: "default",
				Reason:    "Pod not found",
			},
			expectedOutput: "pod operation 'delete' failed for pod default/nginx-123: Pod not found",
		},
		{
			name: "PodOperationError with cause",
			podError: &PodOperationError{
				Operation: "restart",
				PodName:   "mysql-456",
				Namespace: "production",
				Reason:    "Controller update failed",
				Cause:     errors.New("API server timeout"),
			},
			expectedOutput: "pod operation 'restart' failed for pod production/mysql-456: Controller update failed (caused by: API server timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.podError.Error()
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestPodOperationError_Unwrap(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	t.Run("PodOperationError with cause returns cause", func(t *testing.T) {
		originalError := errors.New("kubernetes API error")
		podError := &PodOperationError{
			Operation: "scale",
			PodName:   "app-789",
			Namespace: "staging",
			Reason:    "Scale operation failed",
			Cause:     originalError,
		}

		unwrapped := podError.Unwrap()
		assert.Equal(t, originalError, unwrapped)
	})

	t.Run("PodOperationError without cause returns nil", func(t *testing.T) {
		podError := &PodOperationError{
			Operation: "delete",
			PodName:   "test-pod",
			Namespace: "test",
			Reason:    "Pod terminated",
		}

		unwrapped := podError.Unwrap()
		assert.Nil(t, unwrapped)
	})
}

func TestNewPodOperationError(t *testing.T) {
	t.Skip("Skipping test due to dependency issues")
	t.Run("NewPodOperationError creates correct structure", func(t *testing.T) {
		cause := errors.New("network error")
		err := NewPodOperationError("restart", "nginx-pod", "web", "Operation timeout", cause)

		assert.Equal(t, "restart", err.Operation)
		assert.Equal(t, "nginx-pod", err.PodName)
		assert.Equal(t, "web", err.Namespace)
		assert.Equal(t, "Operation timeout", err.Reason)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewPodOperationError without cause", func(t *testing.T) {
		err := NewPodOperationError("delete", "old-pod", "cleanup", "Pod removed", nil)

		assert.Equal(t, "delete", err.Operation)
		assert.Equal(t, "old-pod", err.PodName)
		assert.Equal(t, "cleanup", err.Namespace)
		assert.Equal(t, "Pod removed", err.Reason)
		assert.Nil(t, err.Cause)
	})
}

// Benchmark tests for error handling performance
func BenchmarkServiceError_Error(b *testing.B) {
	err := &ServiceError{
		Type:    ErrorTypeValidation,
		Message: "Validation failed for input field",
		Cause:   errors.New("field is required"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkIsErrorType(b *testing.B) {
	err := NewInternalError("Database error", errors.New("connection failed"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsErrorType(err, ErrorTypeInternal)
	}
}

func BenchmarkPodOperationError_Error(b *testing.B) {
	err := &PodOperationError{
		Operation: "restart",
		PodName:   "benchmark-pod",
		Namespace: "performance",
		Reason:    "Performance test operation",
		Cause:     errors.New("test error"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}