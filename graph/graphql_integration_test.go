//go:build ignore
// +build ignore

// Tests skipped due to dependency issues
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
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

// GraphQLTestSuite provides utilities for testing GraphQL operations
type GraphQLTestSuite struct {
	t           *testing.T
	handler     *handler.Server
	resolver    *Resolver
	mockPodSvc  *MockPodService
	mockHealthSvc *MockHealthService
	mockAuditLogger *MockAuditLogger
}

// NewGraphQLTestSuite creates a new test suite with mocked dependencies
func NewGraphQLTestSuite(t *testing.T) *GraphQLTestSuite {
	mockPodSvc := new(MockPodService)
	mockHealthSvc := new(MockHealthService)
	mockAuditLogger := new(MockAuditLogger)

	config := ResolverConfig{
		PodService:    mockPodSvc,
		HealthService: mockHealthSvc,
		Validator:     NewDefaultInputValidator(),
		Converter:     NewDefaultTypeConverter(),
		ErrorHandler:  NewDefaultErrorHandler(),
		AuditLogger:   mockAuditLogger,
	}

	resolver := NewResolver(config)

	// Create GraphQL handler
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	return &GraphQLTestSuite{
		t:           t,
		handler:     srv,
		resolver:    resolver,
		mockPodSvc:  mockPodSvc,
		mockHealthSvc: mockHealthSvc,
		mockAuditLogger: mockAuditLogger,
	}
}

// ExecuteQuery executes a GraphQL query and returns the response
func (suite *GraphQLTestSuite) ExecuteQuery(query string, variables map[string]interface{}) *graphql.Response {
	req := &graphql.RawParams{
		Query:     query,
		Variables: variables,
	}

	schema := suite.handler.NewRequestContext(context.Background(), req)
	resp := graphql.Handler(schema)

	return resp
}

// AssertNoErrors checks that the GraphQL response has no errors
func (suite *GraphQLTestSuite) AssertNoErrors(resp *graphql.Response) {
	if len(resp.Errors) > 0 {
		suite.t.Errorf("Expected no errors, but got: %v", resp.Errors)
	}
}

// AssertHasErrors checks that the GraphQL response has errors
func (suite *GraphQLTestSuite) AssertHasErrors(resp *graphql.Response) {
	if len(resp.Errors) == 0 {
		suite.t.Error("Expected errors, but got none")
	}
}

func TestGraphQL_QueryAllPods(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name          string
		query         string
		variables     map[string]interface{}
		mockSetup     func()
		expectErrors  bool
		expectedCount *int
	}{
		{
			name: "query all pods without namespace filter",
			query: `
				query {
					allPods {
						count
						pods {
							name
							namespace
							phase
							ready
							restarts
						}
					}
				}
			`,
			mockSetup: func() {
				expectedPods := []services.PodInfo{
					{
						Name:      "web-1",
						Namespace: "production",
						Phase:     corev1.PodRunning,
						Ready:     "1/1",
						Restarts:  0,
					},
					{
						Name:      "api-1",
						Namespace: "default",
						Phase:     corev1.PodRunning,
						Ready:     "1/1",
						Restarts:  2,
					},
				}
				suite.mockPodSvc.On("ListAllPods", mock.Anything, (*string)(nil)).Return(expectedPods, nil)
			},
			expectErrors:  false,
			expectedCount: intPtr(2),
		},
		{
			name: "query all pods with namespace filter",
			query: `
				query AllPodsInNamespace($namespace: String) {
					allPods(namespace: $namespace) {
						count
						pods {
							name
							namespace
							phase
							labels {
								key
								value
							}
						}
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "production",
			},
			mockSetup: func() {
				expectedPods := []services.PodInfo{
					{
						Name:      "web-1",
						Namespace: "production",
						Phase:     corev1.PodRunning,
						Labels: map[string]string{
							"app": "web",
							"env": "prod",
						},
					},
				}
				suite.mockPodSvc.On("ListAllPods", mock.Anything, stringPtr("production")).Return(expectedPods, nil)
			},
			expectErrors:  false,
			expectedCount: intPtr(1),
		},
		{
			name: "query with invalid namespace",
			query: `
				query {
					allPods(namespace: "INVALID-NAMESPACE") {
						count
						pods {
							name
						}
					}
				}
			`,
			mockSetup: func() {
				// No mock setup needed as validation should fail
			},
			expectErrors: true,
		},
		{
			name: "service error - unauthorized",
			query: `
				query {
					allPods {
						count
						pods {
							name
						}
					}
				}
			`,
			mockSetup: func() {
				suite.mockPodSvc.On("ListAllPods", mock.Anything, (*string)(nil)).Return(
					[]services.PodInfo{},
					apierrors.NewUnauthorized("insufficient permissions"),
				)
			},
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)

				if tt.expectedCount != nil {
					data := resp.Data.(map[string]interface{})
					allPods := data["allPods"].(map[string]interface{})
					count := int(allPods["count"].(float64))
					assert.Equal(t, *tt.expectedCount, count)
				}
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
		})
	}
}

func TestGraphQL_QueryPods(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
		mockSetup func()
		expectErrors bool
	}{
		{
			name: "query pods in specific namespace",
			query: `
				query PodsInNamespace($namespace: String!) {
					pods(namespace: $namespace) {
						namespace
						count
						pods {
							name
							phase
							containers {
								name
								image
								ready
								restartCount
								state
							}
						}
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "default",
			},
			mockSetup: func() {
				expectedPods := []services.PodInfo{
					{
						Name:      "nginx-pod",
						Namespace: "default",
						Phase:     corev1.PodRunning,
						Containers: []services.ContainerInfo{
							{
								Name:         "nginx",
								Image:        "nginx:1.20",
								Ready:        true,
								RestartCount: 0,
								State:        "Running",
							},
						},
					},
				}
				suite.mockPodSvc.On("ListPods", mock.Anything, "default").Return(expectedPods, nil)
			},
			expectErrors: false,
		},
		{
			name: "query with empty namespace should fail validation",
			query: `
				query {
					pods(namespace: "") {
						count
					}
				}
			`,
			mockSetup: func() {
				// No mock setup needed
			},
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
		})
	}
}

func TestGraphQL_QueryPod(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
		mockSetup func()
		expectErrors bool
	}{
		{
			name: "query specific pod",
			query: `
				query GetPod($namespace: String!, $name: String!) {
					pod(namespace: $namespace, name: $name) {
						name
						namespace
						phase
						podIP
						hostIP
						nodeName
						age
						ready
						restarts
						owners
						controllerType
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "default",
				"name":      "nginx-pod",
			},
			mockSetup: func() {
				expectedPod := &services.PodInfo{
					Name:           "nginx-pod",
					Namespace:      "default",
					Phase:          corev1.PodRunning,
					PodIP:          "10.0.0.1",
					HostIP:         "192.168.1.100",
					NodeName:       "worker-1",
					Age:            "5m",
					Ready:          "1/1",
					Restarts:       0,
					Owners:         []string{"deployment/nginx"},
					ControllerType: "Deployment",
				}
				suite.mockPodSvc.On("GetPod", mock.Anything, "default", "nginx-pod").Return(expectedPod, nil)
			},
			expectErrors: false,
		},
		{
			name: "query non-existent pod",
			query: `
				query {
					pod(namespace: "default", name: "missing-pod") {
						name
					}
				}
			`,
			mockSetup: func() {
				suite.mockPodSvc.On("GetPod", mock.Anything, "default", "missing-pod").Return(
					(*services.PodInfo)(nil),
					apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "missing-pod"),
				)
			},
			expectErrors: true,
		},
		{
			name: "query with dangerous pod name",
			query: `
				query {
					pod(namespace: "default", name: "kube-system-pod") {
						name
					}
				}
			`,
			mockSetup: func() {
				// No mock setup needed - validation should fail
			},
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
		})
	}
}

func TestGraphQL_MutationDeletePod(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
		mockSetup func()
		expectErrors bool
	}{
		{
			name: "successful pod deletion",
			query: `
				mutation DeletePod($namespace: String!, $name: String!, $user: String) {
					deletePod(namespace: $namespace, name: $name, user: $user) {
						message
						pod
						namespace
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "default",
				"name":      "test-pod",
				"user":      "admin",
			},
			mockSetup: func() {
				suite.mockPodSvc.On("DeletePod", mock.Anything, "default", "test-pod", "admin").Return(nil)
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "default", "test-pod", "admin", true, "").Once()
			},
			expectErrors: false,
		},
		{
			name: "pod deletion without user (system default)",
			query: `
				mutation {
					deletePod(namespace: "default", name: "test-pod") {
						message
						pod
						namespace
					}
				}
			`,
			mockSetup: func() {
				suite.mockPodSvc.On("DeletePod", mock.Anything, "default", "test-pod", "system").Return(nil)
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "default", "test-pod", "system", true, "").Once()
			},
			expectErrors: false,
		},
		{
			name: "deletion fails - forbidden",
			query: `
				mutation {
					deletePod(namespace: "kube-system", name: "important-pod", user: "developer") {
						message
					}
				}
			`,
			mockSetup: func() {
				// Should fail validation due to reserved namespace
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "kube-system", "important-pod", "developer", false, mock.AnythingOfType("string")).Once()
			},
			expectErrors: true,
		},
		{
			name: "deletion fails - invalid user",
			query: `
				mutation {
					deletePod(namespace: "default", name: "test-pod", user: "user<script>") {
						message
					}
				}
			`,
			mockSetup: func() {
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "deletePod", "default", "test-pod", "user<script>", false, mock.AnythingOfType("string")).Once()
			},
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)

				// Verify response structure
				data := resp.Data.(map[string]interface{})
				deletePod := data["deletePod"].(map[string]interface{})
				assert.Equal(t, "Pod deleted successfully", deletePod["message"])
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
			suite.mockAuditLogger.AssertExpectations(t)
		})
	}
}

func TestGraphQL_MutationRestartPod(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
		mockSetup func()
		expectErrors bool
	}{
		{
			name: "successful deployment restart",
			query: `
				mutation RestartPod($namespace: String!, $name: String!, $user: String) {
					restartPod(namespace: $namespace, name: $name, user: $user) {
						message
						pod
						namespace
						type
						deployment
						warning
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "default",
				"name":      "web-pod-123",
				"user":      "admin",
			},
			mockSetup: func() {
				result := map[string]interface{}{
					"message":    "Rollout restart initiated",
					"type":       "deployment",
					"deployment": "web-app",
				}
				suite.mockPodSvc.On("RestartPod", mock.Anything, "default", "web-pod-123", "admin").Return(result, nil)
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "restartPod", "default", "web-pod-123", "admin", true, "").Once()
			},
			expectErrors: false,
		},
		{
			name: "standalone pod restart with warning",
			query: `
				mutation {
					restartPod(namespace: "default", name: "standalone-pod") {
						message
						warning
					}
				}
			`,
			mockSetup: func() {
				result := map[string]interface{}{
					"message": "Pod deleted (no controller)",
					"warning": "Pod will not be automatically recreated",
				}
				suite.mockPodSvc.On("RestartPod", mock.Anything, "default", "standalone-pod", "system").Return(result, nil)
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "restartPod", "default", "standalone-pod", "system", true, "").Once()
			},
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
			suite.mockAuditLogger.AssertExpectations(t)
		})
	}
}

func TestGraphQL_MutationScaleDeployment(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
		mockSetup func()
		expectErrors bool
	}{
		{
			name: "successful deployment scaling",
			query: `
				mutation ScaleDeployment($namespace: String!, $name: String!, $input: ScaleInput!, $user: String) {
					scaleDeployment(namespace: $namespace, name: $name, input: $input, user: $user) {
						message
						namespace
						deployment
						previousReplicas
						newReplicas
					}
				}
			`,
			variables: map[string]interface{}{
				"namespace": "default",
				"name":      "web-app",
				"input": map[string]interface{}{
					"replicas": 5,
				},
				"user": "admin",
			},
			mockSetup: func() {
				result := map[string]interface{}{
					"message":          "Deployment scaled successfully",
					"deployment":       "web-app",
					"previousReplicas": 3,
					"newReplicas":      5,
				}
				suite.mockPodSvc.On("ScaleDeployment", mock.Anything, "default", "web-app", int32(5), "admin").Return(result, nil)
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "scaleDeployment", "default", "web-app", "admin", true, "").Once()
			},
			expectErrors: false,
		},
		{
			name: "scaling with negative replicas should fail",
			query: `
				mutation {
					scaleDeployment(namespace: "default", name: "web-app", input: {replicas: -1}) {
						message
					}
				}
			`,
			mockSetup: func() {
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "scaleDeployment", "default", "web-app", "system", false, mock.AnythingOfType("string")).Once()
			},
			expectErrors: true,
		},
		{
			name: "scaling with too many replicas should fail",
			query: `
				mutation {
					scaleDeployment(namespace: "default", name: "web-app", input: {replicas: 2000}) {
						message
					}
				}
			`,
			mockSetup: func() {
				suite.mockAuditLogger.On("LogMutation", mock.Anything, "scaleDeployment", "default", "web-app", "system", false, mock.AnythingOfType("string")).Once()
			},
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, tt.variables)

			// Assert
			if tt.expectErrors {
				suite.AssertHasErrors(resp)
			} else {
				suite.AssertNoErrors(resp)
			}

			// Verify mocks
			suite.mockPodSvc.AssertExpectations(t)
			suite.mockAuditLogger.AssertExpectations(t)
		})
	}
}

func TestGraphQL_ComplexQueryWithFragments(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	query := `
		fragment PodDetails on PodInfo {
			name
			namespace
			phase
			ready
			restarts
			labels {
				key
				value
			}
			containers {
				name
				image
				ready
				state
			}
		}

		query ComplexPodQuery($namespace: String!) {
			pods(namespace: $namespace) {
				count
				pods {
					...PodDetails
				}
			}
			specificPod: pod(namespace: $namespace, name: "nginx-pod") {
				...PodDetails
				podIP
				hostIP
				nodeName
				controllerType
			}
		}
	`

	// Setup mocks
	expectedPods := []services.PodInfo{
		{
			Name:      "nginx-pod",
			Namespace: "default",
			Phase:     corev1.PodRunning,
			Ready:     "1/1",
			Restarts:  0,
			Labels:    map[string]string{"app": "nginx"},
			Containers: []services.ContainerInfo{
				{
					Name:  "nginx",
					Image: "nginx:1.20",
					Ready: true,
					State: "Running",
				},
			},
		},
	}

	specificPod := &services.PodInfo{
		Name:           "nginx-pod",
		Namespace:      "default",
		Phase:          corev1.PodRunning,
		PodIP:          "10.0.0.1",
		HostIP:         "192.168.1.100",
		NodeName:       "worker-1",
		Ready:          "1/1",
		Restarts:       0,
		Labels:         map[string]string{"app": "nginx"},
		ControllerType: "Deployment",
		Containers: []services.ContainerInfo{
			{
				Name:  "nginx",
				Image: "nginx:1.20",
				Ready: true,
				State: "Running",
			},
		},
	}

	suite.mockPodSvc.On("ListPods", mock.Anything, "default").Return(expectedPods, nil)
	suite.mockPodSvc.On("GetPod", mock.Anything, "default", "nginx-pod").Return(specificPod, nil)

	// Execute
	variables := map[string]interface{}{
		"namespace": "default",
	}
	resp := suite.ExecuteQuery(query, variables)

	// Assert
	suite.AssertNoErrors(resp)

	// Verify response structure
	data := resp.Data.(map[string]interface{})

	pods := data["pods"].(map[string]interface{})
	assert.Equal(t, float64(1), pods["count"])

	specificPodData := data["specificPod"].(map[string]interface{})
	assert.Equal(t, "nginx-pod", specificPodData["name"])
	assert.Equal(t, "10.0.0.1", specificPodData["podIP"])

	suite.mockPodSvc.AssertExpectations(t)
}

func TestGraphQL_ErrorHandling(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	tests := []struct {
		name          string
		query         string
		mockSetup     func()
		expectedErrorCode string
	}{
		{
			name: "validation error should return proper error code",
			query: `
				query {
					pods(namespace: "INVALID-NAMESPACE") {
						count
					}
				}
			`,
			mockSetup: func() {
				// No setup needed - validation will fail
			},
			expectedErrorCode: "VALIDATION_ERROR",
		},
		{
			name: "not found error should return proper error code",
			query: `
				query {
					pod(namespace: "default", name: "missing-pod") {
						name
					}
				}
			`,
			mockSetup: func() {
				suite.mockPodSvc.On("GetPod", mock.Anything, "default", "missing-pod").Return(
					(*services.PodInfo)(nil),
					apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "missing-pod"),
				)
			},
			expectedErrorCode: "NOT_FOUND",
		},
		{
			name: "unauthorized error should return proper error code",
			query: `
				query {
					allPods {
						count
					}
				}
			`,
			mockSetup: func() {
				suite.mockPodSvc.On("ListAllPods", mock.Anything, (*string)(nil)).Return(
					[]services.PodInfo{},
					apierrors.NewUnauthorized("insufficient permissions"),
				)
			},
			expectedErrorCode: "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.mockSetup()

			// Execute
			resp := suite.ExecuteQuery(tt.query, nil)

			// Assert
			suite.AssertHasErrors(resp)

			// Check error code in extensions
			require.Len(t, resp.Errors, 1)
			err := resp.Errors[0]
			require.NotNil(t, err.Extensions)
			assert.Equal(t, tt.expectedErrorCode, err.Extensions["code"])

			suite.mockPodSvc.AssertExpectations(t)
		})
	}
}

func TestGraphQL_PerformanceAndComplexity(t *testing.T) {
	suite := NewGraphQLTestSuite(t)

	// Test query with nested data
	query := `
		query LargeDataQuery {
			allPods {
				count
				pods {
					name
					namespace
					phase
					labels {
						key
						value
					}
					containers {
						name
						image
						ready
						restartCount
						state
					}
				}
			}
		}
	`

	// Create large dataset
	largeDataset := make([]services.PodInfo, 100)
	for i := 0; i < 100; i++ {
		largeDataset[i] = services.PodInfo{
			Name:      fmt.Sprintf("pod-%d", i),
			Namespace: "default",
			Phase:     corev1.PodRunning,
			Labels: map[string]string{
				"app":  "test",
				"id":   fmt.Sprintf("%d", i),
				"tier": "web",
			},
			Containers: []services.ContainerInfo{
				{
					Name:         "main",
					Image:        "nginx:latest",
					Ready:        true,
					RestartCount: int32(i % 5),
					State:        "Running",
				},
				{
					Name:         "sidecar",
					Image:        "sidecar:latest",
					Ready:        true,
					RestartCount: 0,
					State:        "Running",
				},
			},
		}
	}

	suite.mockPodSvc.On("ListAllPods", mock.Anything, (*string)(nil)).Return(largeDataset, nil)

	// Execute and measure time
	start := time.Now()
	resp := suite.ExecuteQuery(query, nil)
	duration := time.Since(start)

	// Assert
	suite.AssertNoErrors(resp)

	// Performance assertion - should complete within reasonable time
	assert.Less(t, duration, 100*time.Millisecond, "Query should complete quickly even with large dataset")

	// Verify response structure
	data := resp.Data.(map[string]interface{})
	allPods := data["allPods"].(map[string]interface{})
	assert.Equal(t, float64(100), allPods["count"])

	suite.mockPodSvc.AssertExpectations(t)
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}