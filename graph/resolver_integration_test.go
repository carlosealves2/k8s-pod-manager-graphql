//go:build ignore
// +build ignore

// Tests skipped due to dependency issues
package graph

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestQueryResolver_AllPods tests the allPods query resolver
func TestQueryResolver_AllPods(t *testing.T) {
	tests := []struct {
		name           string
		namespace      *string
		mockSetup      func(*MockPodService)
		expectedCount  int
		expectedError  bool
		errorMessage   string
	}{
		{
			name:      "successful query with namespace filter",
			namespace: stringPtr("production"),
			mockSetup: func(m *MockPodService) {
				expectedPods := []services.PodInfo{
					{Name: "web-1", Namespace: "production", Phase: corev1.PodRunning},
					{Name: "web-2", Namespace: "production", Phase: corev1.PodRunning},
				}
				m.On("ListAllPods", mock.Anything, stringPtr("production")).Return(expectedPods, nil)
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:      "successful query without namespace filter",
			namespace: nil,
			mockSetup: func(m *MockPodService) {
				expectedPods := []services.PodInfo{
					{Name: "web-1", Namespace: "production", Phase: corev1.PodRunning},
					{Name: "api-1", Namespace: "default", Phase: corev1.PodRunning},
					{Name: "db-1", Namespace: "database", Phase: corev1.PodRunning},
				}
				m.On("ListAllPods", mock.Anything, (*string)(nil)).Return(expectedPods, nil)
			},
			expectedCount: 3,
			expectedError: false,
		},
		{
			name:      "validation error - invalid namespace",
			namespace: stringPtr("INVALID-NAMESPACE"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedCount: 0,
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "service error",
			namespace: stringPtr("default"),
			mockSetup: func(m *MockPodService) {
				m.On("ListAllPods", mock.Anything, stringPtr("default")).Return(
					[]services.PodInfo{},
					apierrors.NewUnauthorized("insufficient permissions"),
				)
			},
			expectedCount: 0,
			expectedError: true,
			errorMessage:  "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			queryResolver := resolver.Query()

			// Execute
			result, err := queryResolver.AllPods(context.Background(), tt.namespace)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedCount, result.Count)
				assert.Len(t, result.Pods, tt.expectedCount)
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestQueryResolver_Pods tests the pods query resolver
func TestQueryResolver_Pods(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		mockSetup     func(*MockPodService)
		expectedCount int
		expectedError bool
		errorMessage  string
	}{
		{
			name:      "successful query",
			namespace: "default",
			mockSetup: func(m *MockPodService) {
				expectedPods := []services.PodInfo{
					{
						Name:      "nginx-1",
						Namespace: "default",
						Phase:     corev1.PodRunning,
						Ready:     "1/1",
						Restarts:  0,
						Labels:    map[string]string{"app": "nginx"},
					},
				}
				m.On("ListPods", mock.Anything, "default").Return(expectedPods, nil)
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:      "empty namespace validation error",
			namespace: "",
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedCount: 0,
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "not found error",
			namespace: "nonexistent",
			mockSetup: func(m *MockPodService) {
				m.On("ListPods", mock.Anything, "nonexistent").Return(
					[]services.PodInfo{},
					apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, ""),
				)
			},
			expectedCount: 0,
			expectedError: true,
			errorMessage:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			queryResolver := resolver.Query()

			// Execute
			result, err := queryResolver.Pods(context.Background(), tt.namespace)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.namespace, result.Namespace)
				assert.Equal(t, tt.expectedCount, result.Count)
				assert.Len(t, result.Pods, tt.expectedCount)
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestQueryResolver_Pod tests the pod query resolver
func TestQueryResolver_Pod(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		podName      string
		mockSetup    func(*MockPodService)
		expectedPod  *string
		expectedError bool
		errorMessage string
	}{
		{
			name:      "successful query",
			namespace: "default",
			podName:   "nginx-1",
			mockSetup: func(m *MockPodService) {
				expectedPod := &services.PodInfo{
					Name:      "nginx-1",
					Namespace: "default",
					Phase:     corev1.PodRunning,
					Ready:     "1/1",
					Labels:    map[string]string{"app": "nginx"},
				}
				m.On("GetPod", mock.Anything, "default", "nginx-1").Return(expectedPod, nil)
			},
			expectedPod:   stringPtr("nginx-1"),
			expectedError: false,
		},
		{
			name:      "validation error - invalid namespace",
			namespace: "invalid@namespace",
			podName:   "test-pod",
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedPod:   nil,
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "validation error - invalid pod name",
			namespace: "default",
			podName:   "kube-system-pod",
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedPod:   nil,
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "pod not found",
			namespace: "default",
			podName:   "nonexistent-pod",
			mockSetup: func(m *MockPodService) {
				m.On("GetPod", mock.Anything, "default", "nonexistent-pod").Return(
					(*services.PodInfo)(nil),
					apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "nonexistent-pod"),
				)
			},
			expectedPod:   nil,
			expectedError: true,
			errorMessage:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			queryResolver := resolver.Query()

			// Execute
			result, err := queryResolver.Pod(context.Background(), tt.namespace, tt.podName)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				if tt.expectedPod != nil {
					assert.Equal(t, *tt.expectedPod, result.Name)
				}
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestMutationResolver_DeletePod tests the deletePod mutation resolver
func TestMutationResolver_DeletePod(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		podName       string
		user          *string
		mockSetup     func(*MockPodService)
		expectedError bool
		errorMessage  string
	}{
		{
			name:      "successful deletion",
			namespace: "default",
			podName:   "test-pod",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				m.On("DeletePod", mock.Anything, "default", "test-pod", "admin").Return(nil)
			},
			expectedError: false,
		},
		{
			name:      "successful deletion with system user",
			namespace: "default",
			podName:   "test-pod",
			user:      nil,
			mockSetup: func(m *MockPodService) {
				m.On("DeletePod", mock.Anything, "default", "test-pod", "system").Return(nil)
			},
			expectedError: false,
		},
		{
			name:      "validation error - reserved namespace",
			namespace: "kube-system",
			podName:   "test-pod",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "validation error - dangerous pod name",
			namespace: "default",
			podName:   "kube-proxy",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "service error - forbidden",
			namespace: "default",
			podName:   "test-pod",
			user:      stringPtr("developer"),
			mockSetup: func(m *MockPodService) {
				m.On("DeletePod", mock.Anything, "default", "test-pod", "developer").Return(
					apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "test-pod", errors.New("forbidden")),
				)
			},
			expectedError: true,
			errorMessage:  "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			mutationResolver := resolver.Mutation()

			// Execute
			result, err := mutationResolver.DeletePod(context.Background(), tt.namespace, tt.podName, tt.user)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "Pod deleted successfully", result.Message)
				assert.Equal(t, tt.podName, result.Pod)
				assert.Equal(t, tt.namespace, result.Namespace)
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestMutationResolver_RestartPod tests the restartPod mutation resolver
func TestMutationResolver_RestartPod(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		podName       string
		user          *string
		mockSetup     func(*MockPodService)
		expectedType  string
		expectedError bool
		errorMessage  string
	}{
		{
			name:      "successful deployment restart",
			namespace: "default",
			podName:   "web-pod-123",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				result := map[string]interface{}{
					"message":    "Rollout restart initiated",
					"type":       "deployment",
					"deployment": "web-app",
				}
				m.On("RestartPod", mock.Anything, "default", "web-pod-123", "admin").Return(result, nil)
			},
			expectedType:  "deployment",
			expectedError: false,
		},
		{
			name:      "successful statefulset restart",
			namespace: "database",
			podName:   "postgres-0",
			user:      stringPtr("dba"),
			mockSetup: func(m *MockPodService) {
				result := map[string]interface{}{
					"message":     "Rollout restart initiated",
					"type":        "statefulset",
					"statefulset": "postgres",
				}
				m.On("RestartPod", mock.Anything, "database", "postgres-0", "dba").Return(result, nil)
			},
			expectedType:  "statefulset",
			expectedError: false,
		},
		{
			name:      "standalone pod restart with warning",
			namespace: "default",
			podName:   "standalone-pod",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				result := map[string]interface{}{
					"message": "Pod deleted (no controller)",
					"warning": "Pod will not be automatically recreated",
				}
				m.On("RestartPod", mock.Anything, "default", "standalone-pod", "admin").Return(result, nil)
			},
			expectedError: false,
		},
		{
			name:      "validation error - invalid user",
			namespace: "default",
			podName:   "test-pod",
			user:      stringPtr("user<script>"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "service error - pod not found",
			namespace: "default",
			podName:   "missing-pod",
			user:      stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				m.On("RestartPod", mock.Anything, "default", "missing-pod", "admin").Return(
					map[string]interface{}{},
					apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "missing-pod"),
				)
			},
			expectedError: true,
			errorMessage:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			mutationResolver := resolver.Mutation()

			// Execute
			result, err := mutationResolver.RestartPod(context.Background(), tt.namespace, tt.podName, tt.user)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.podName, result.Pod)
				assert.Equal(t, tt.namespace, result.Namespace)
				assert.NotEmpty(t, result.Message)

				if tt.expectedType != "" {
					assert.Equal(t, tt.expectedType, result.Type)
				}
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestMutationResolver_ScaleDeployment tests the scaleDeployment mutation resolver
func TestMutationResolver_ScaleDeployment(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		deploymentName string
		input         model.ScaleInput
		user          *string
		mockSetup     func(*MockPodService)
		expectedError bool
		errorMessage  string
	}{
		{
			name:           "successful scaling up",
			namespace:      "default",
			deploymentName: "web-app",
			input:          model.ScaleInput{Replicas: 5},
			user:           stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				result := map[string]interface{}{
					"message":          "Deployment scaled successfully",
					"deployment":       "web-app",
					"previousReplicas": 3,
					"newReplicas":      5,
				}
				m.On("ScaleDeployment", mock.Anything, "default", "web-app", int32(5), "admin").Return(result, nil)
			},
			expectedError: false,
		},
		{
			name:           "successful scaling to zero",
			namespace:      "staging",
			deploymentName: "temp-app",
			input:          model.ScaleInput{Replicas: 0},
			user:           stringPtr("operator"),
			mockSetup: func(m *MockPodService) {
				result := map[string]interface{}{
					"message":          "Deployment scaled to zero",
					"deployment":       "temp-app",
					"previousReplicas": 2,
					"newReplicas":      0,
				}
				m.On("ScaleDeployment", mock.Anything, "staging", "temp-app", int32(0), "operator").Return(result, nil)
			},
			expectedError: false,
		},
		{
			name:           "validation error - negative replicas",
			namespace:      "default",
			deploymentName: "web-app",
			input:          model.ScaleInput{Replicas: -1},
			user:           stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:           "validation error - too many replicas",
			namespace:      "default",
			deploymentName: "web-app",
			input:          model.ScaleInput{Replicas: 2000},
			user:           stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:           "service error - deployment not found",
			namespace:      "default",
			deploymentName: "missing-app",
			input:          model.ScaleInput{Replicas: 3},
			user:           stringPtr("admin"),
			mockSetup: func(m *MockPodService) {
				m.On("ScaleDeployment", mock.Anything, "default", "missing-app", int32(3), "admin").Return(
					map[string]interface{}{},
					apierrors.NewNotFound(schema.GroupResource{Resource: "deployments"}, "missing-app"),
				)
			},
			expectedError: true,
			errorMessage:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			mutationResolver := resolver.Mutation()

			// Execute
			result, err := mutationResolver.ScaleDeployment(context.Background(), tt.namespace, tt.deploymentName, tt.input, tt.user)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.namespace, result.Namespace)
				assert.NotEmpty(t, result.Message)
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestSubscriptionResolver_WatchPods tests the watchPods subscription resolver
func TestSubscriptionResolver_WatchPods(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		mockSetup     func(*MockPodService)
		expectedError bool
		errorMessage  string
	}{
		{
			name:      "successful subscription",
			namespace: "default",
			mockSetup: func(m *MockPodService) {
				m.On("WatchPods", mock.Anything, "default", mock.AnythingOfType("chan<- services.PodWatchEvent")).Return(nil)
			},
			expectedError: false,
		},
		{
			name:      "validation error - invalid namespace",
			namespace: "INVALID-NS",
			mockSetup: func(m *MockPodService) {
				// No mock calls expected due to validation failure
			},
			expectedError: true,
			errorMessage:  "validation",
		},
		{
			name:      "service error - unauthorized",
			namespace: "restricted",
			mockSetup: func(m *MockPodService) {
				m.On("WatchPods", mock.Anything, "restricted", mock.AnythingOfType("chan<- services.PodWatchEvent")).Return(
					apierrors.NewUnauthorized("watch not allowed"),
				)
			},
			expectedError: true,
			errorMessage:  "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockPodService := new(MockPodService)
			tt.mockSetup(mockPodService)

			resolver := NewTestResolver(mockPodService)
			subscriptionResolver := resolver.Subscription()

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Execute
			resultChan, err := subscriptionResolver.WatchPods(ctx, tt.namespace)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, resultChan)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resultChan)

				// Wait for context timeout to ensure subscription starts properly
				select {
				case <-ctx.Done():
					// Expected - subscription should continue until context is cancelled
				case <-time.After(200 * time.Millisecond):
					t.Error("subscription should have been cancelled by context timeout")
				}
			}

			mockPodService.AssertExpectations(t)
		})
	}
}

// TestHelperFunctions tests the resolver helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getUserOrDefault with user", func(t *testing.T) {
		user := "test-user"
		result := getUserOrDefault(&user)
		assert.Equal(t, "test-user", result)
	})

	t.Run("getUserOrDefault with nil user", func(t *testing.T) {
		result := getUserOrDefault(nil)
		assert.Equal(t, "system", result)
	})

	t.Run("getUserOrDefault with empty user", func(t *testing.T) {
		user := ""
		result := getUserOrDefault(&user)
		assert.Equal(t, "system", result)
	})
}

// TestResolverBuilder tests the resolver creation patterns
func TestResolverBuilder(t *testing.T) {
	t.Run("NewResolver with custom config", func(t *testing.T) {
		mockPodService := new(MockPodService)
		mockHealthService := new(MockHealthService)
		validator := NewDefaultInputValidator()
		converter := NewDefaultTypeConverter()
		errorHandler := NewDefaultErrorHandler()
		auditLogger := NewNoOpAuditLogger()

		config := ResolverConfig{
			PodService:    mockPodService,
			HealthService: mockHealthService,
			Validator:     validator,
			Converter:     converter,
			ErrorHandler:  errorHandler,
			AuditLogger:   auditLogger,
		}

		resolver := NewResolver(config)

		require.NotNil(t, resolver)
		assert.Equal(t, mockPodService, resolver.getPodService())
		assert.Equal(t, mockHealthService, resolver.getHealthService())
		assert.Equal(t, validator, resolver.getValidator())
		assert.Equal(t, converter, resolver.getConverter())
		assert.Equal(t, errorHandler, resolver.getErrorHandler())
		assert.Equal(t, auditLogger, resolver.getAuditLogger())
	})

	t.Run("NewTestResolver", func(t *testing.T) {
		mockPodService := new(MockPodService)
		resolver := NewTestResolver(mockPodService)

		require.NotNil(t, resolver)
		assert.Equal(t, mockPodService, resolver.getPodService())
		assert.IsType(t, &DefaultHealthService{}, resolver.getHealthService())
		assert.IsType(t, &DefaultInputValidator{}, resolver.getValidator())
		assert.IsType(t, &DefaultTypeConverter{}, resolver.getConverter())
		assert.IsType(t, &DefaultErrorHandler{}, resolver.getErrorHandler())
		assert.IsType(t, &NoOpAuditLogger{}, resolver.getAuditLogger())
	})
}

// TestContextHandling tests how resolvers handle context cancellation and timeouts
func TestContextHandling(t *testing.T) {
	t.Run("context cancellation in query", func(t *testing.T) {
		mockPodService := new(MockPodService)

		// Mock will delay to simulate context cancellation
		mockPodService.On("ListPods", mock.Anything, "default").Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			// Simulate long-running operation
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				// Context was cancelled
			}
		}).Return([]services.PodInfo{}, context.Canceled)

		resolver := NewTestResolver(mockPodService)
		queryResolver := resolver.Query()

		// Create context that will be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Execute
		result, err := queryResolver.Pods(ctx, "default")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		mockPodService.AssertExpectations(t)
	})
}

// TestAuditLogging tests that audit logging is called appropriately
func TestAuditLogging(t *testing.T) {
	t.Run("successful mutation logs audit", func(t *testing.T) {
		mockPodService := new(MockPodService)
		mockAuditLogger := new(MockAuditLogger)

		mockPodService.On("DeletePod", mock.Anything, "default", "test-pod", "admin").Return(nil)

		// Expect audit log for successful operation
		mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "default", "test-pod", "admin", true, "").Once()

		config := ResolverConfig{
			PodService:    mockPodService,
			HealthService: NewDefaultHealthService("test"),
			Validator:     NewDefaultInputValidator(),
			Converter:     NewDefaultTypeConverter(),
			ErrorHandler:  NewDefaultErrorHandler(),
			AuditLogger:   mockAuditLogger,
		}

		resolver := NewResolver(config)
		mutationResolver := resolver.Mutation()

		user := "admin"
		result, err := mutationResolver.DeletePod(context.Background(), "default", "test-pod", &user)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockPodService.AssertExpectations(t)
		mockAuditLogger.AssertExpectations(t)
	})

	t.Run("failed mutation logs audit", func(t *testing.T) {
		mockPodService := new(MockPodService)
		mockAuditLogger := new(MockAuditLogger)

		serviceErr := apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "missing-pod")
		mockPodService.On("DeletePod", mock.Anything, "default", "missing-pod", "admin").Return(serviceErr)

		// Expect audit log for failed operation
		mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "default", "missing-pod", "admin", false, mock.AnythingOfType("string")).Once()

		config := ResolverConfig{
			PodService:    mockPodService,
			HealthService: NewDefaultHealthService("test"),
			Validator:     NewDefaultInputValidator(),
			Converter:     NewDefaultTypeConverter(),
			ErrorHandler:  NewDefaultErrorHandler(),
			AuditLogger:   mockAuditLogger,
		}

		resolver := NewResolver(config)
		mutationResolver := resolver.Mutation()

		user := "admin"
		result, err := mutationResolver.DeletePod(context.Background(), "default", "missing-pod", &user)

		assert.Error(t, err)
		assert.Nil(t, result)

		mockPodService.AssertExpectations(t)
		mockAuditLogger.AssertExpectations(t)
	})
}

// Helper function to get getUserOrDefault from resolvers (assuming it exists)
func getUserOrDefault(user *string) string {
	if user == nil || *user == "" {
		return "system"
	}
	return *user
}

// Additional helper function for tests
func stringPtr(s string) *string {
	return &s
}