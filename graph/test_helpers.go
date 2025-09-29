//go:build ignore
// +build ignore

package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestDataBuilder provides fluent API for building test data
type TestDataBuilder struct{}

// NewTestDataBuilder creates a new test data builder
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{}
}

// CreatePodInfo creates a services.PodInfo with sensible defaults for testing
func (b *TestDataBuilder) CreatePodInfo(name, namespace string) services.PodInfo {
	return services.PodInfo{
		Name:           name,
		Namespace:      namespace,
		Phase:          corev1.PodRunning,
		PodIP:          "10.0.0.1",
		HostIP:         "192.168.1.100",
		NodeName:       "worker-1",
		Age:            "5m",
		Ready:          "1/1",
		Restarts:       0,
		Owners:         []string{fmt.Sprintf("deployment/%s-deployment", name)},
		ControllerType: "Deployment",
		Containers: []services.ContainerInfo{
			{
				Name:         name,
				Image:        fmt.Sprintf("%s:latest", name),
				Ready:        true,
				RestartCount: 0,
				State:        "Running",
			},
		},
		Labels: map[string]string{
			"app":     name,
			"version": "1.0",
		},
	}
}

// CreatePodInfoWithPhase creates a pod info with specific phase
func (b *TestDataBuilder) CreatePodInfoWithPhase(name, namespace string, phase corev1.PodPhase) services.PodInfo {
	pod := b.CreatePodInfo(name, namespace)
	pod.Phase = phase

	switch phase {
	case corev1.PodPending:
		pod.Ready = "0/1"
		for i := range pod.Containers {
			pod.Containers[i].Ready = false
			pod.Containers[i].State = "Pending"
		}
	case corev1.PodFailed:
		pod.Ready = "0/1"
		pod.Restarts = 3
		for i := range pod.Containers {
			pod.Containers[i].Ready = false
			pod.Containers[i].State = "Failed"
			pod.Containers[i].RestartCount = 3
		}
	case corev1.PodSucceeded:
		pod.Ready = "1/1"
		for i := range pod.Containers {
			pod.Containers[i].State = "Completed"
		}
	}

	return pod
}

// CreatePodInfoWithMultipleContainers creates a pod with multiple containers
func (b *TestDataBuilder) CreatePodInfoWithMultipleContainers(name, namespace string, containerCount int) services.PodInfo {
	pod := b.CreatePodInfo(name, namespace)
	pod.Ready = fmt.Sprintf("%d/%d", containerCount, containerCount)

	containers := make([]services.ContainerInfo, containerCount)
	for i := 0; i < containerCount; i++ {
		containers[i] = services.ContainerInfo{
			Name:         fmt.Sprintf("%s-container-%d", name, i),
			Image:        fmt.Sprintf("container%d:latest", i),
			Ready:        true,
			RestartCount: int32(i),
			State:        "Running",
		}
	}
	pod.Containers = containers

	return pod
}

// CreateNamespaceInfo creates a services.NamespaceInfo for testing
func (b *TestDataBuilder) CreateNamespaceInfo(name string) services.NamespaceInfo {
	return services.NamespaceInfo{
		Name:   name,
		Status: "Active",
		Age:    "30d",
		Labels: map[string]string{
			"name": name,
			"tier": "production",
		},
		Annotations: map[string]string{
			"description":         fmt.Sprintf("%s namespace", name),
			"kubernetes.io/owner": "platform-team",
		},
	}
}

// CreateDeployment creates an appsv1.Deployment for testing
func (b *TestDataBuilder) CreateDeployment(name, namespace string, replicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-24 * time.Hour)),
			Labels: map[string]string{
				"app":     name,
				"version": "1.0",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:   replicas,
			ReadyReplicas:     replicas,
			AvailableReplicas: replicas,
		},
	}
}

// CreateStatefulSet creates an appsv1.StatefulSet for testing
func (b *TestDataBuilder) CreateStatefulSet(name, namespace string, replicas int32) appsv1.StatefulSet {
	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-12 * time.Hour)),
			Labels: map[string]string{
				"app":  name,
				"type": "statefulset",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   replicas - 1, // Simulate rolling update
			CurrentReplicas: replicas,
			UpdatedReplicas: replicas - 1,
		},
	}
}

// CreatePodWatchEvent creates a services.PodWatchEvent for testing
func (b *TestDataBuilder) CreatePodWatchEvent(eventType string, pod services.PodInfo, reason string) services.PodWatchEvent {
	event := services.PodWatchEvent{
		Type: eventType,
		Pod:  pod,
	}

	if reason != "" {
		event.Reason = reason
	}

	return event
}

// ErrorBuilder provides utilities for creating test errors
type ErrorBuilder struct{}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder() *ErrorBuilder {
	return &ErrorBuilder{}
}

// NotFound creates a Kubernetes NotFound error
func (e *ErrorBuilder) NotFound(resource, name string) error {
	return apierrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
}

// Unauthorized creates a Kubernetes Unauthorized error
func (e *ErrorBuilder) Unauthorized(message string) error {
	return apierrors.NewUnauthorized(message)
}

// Forbidden creates a Kubernetes Forbidden error
func (e *ErrorBuilder) Forbidden(resource, name, message string) error {
	return apierrors.NewForbidden(schema.GroupResource{Resource: resource}, name, fmt.Errorf(message))
}

// AlreadyExists creates a Kubernetes AlreadyExists error
func (e *ErrorBuilder) AlreadyExists(resource, name string) error {
	return apierrors.NewAlreadyExists(schema.GroupResource{Resource: resource}, name)
}

// Timeout creates a Kubernetes Timeout error
func (e *ErrorBuilder) Timeout(message string, retryAfter int) error {
	return apierrors.NewTimeoutError(message, retryAfter)
}

// MockSetupHelper provides utilities for setting up mocks consistently
type MockSetupHelper struct {
	builder *TestDataBuilder
	errors  *ErrorBuilder
}

// NewMockSetupHelper creates a new mock setup helper
func NewMockSetupHelper() *MockSetupHelper {
	return &MockSetupHelper{
		builder: NewTestDataBuilder(),
		errors:  NewErrorBuilder(),
	}
}

// SetupSuccessfulListPods sets up mock for successful pod listing
func (h *MockSetupHelper) SetupSuccessfulListPods(mockSvc *MockPodService, namespace string, podCount int) []services.PodInfo {
	pods := make([]services.PodInfo, podCount)
	for i := 0; i < podCount; i++ {
		pods[i] = h.builder.CreatePodInfo(fmt.Sprintf("pod-%d", i), namespace)
	}

	mockSvc.On("ListPods", mock.Anything, namespace).Return(pods, nil)
	return pods
}

// SetupSuccessfulListAllPods sets up mock for successful all pods listing
func (h *MockSetupHelper) SetupSuccessfulListAllPods(mockSvc *MockPodService, namespace *string, podCount int) []services.PodInfo {
	pods := make([]services.PodInfo, podCount)
	for i := 0; i < podCount; i++ {
		ns := "default"
		if namespace != nil {
			ns = *namespace
		}
		pods[i] = h.builder.CreatePodInfo(fmt.Sprintf("pod-%d", i), ns)
	}

	mockSvc.On("ListAllPods", mock.Anything, namespace).Return(pods, nil)
	return pods
}

// SetupSuccessfulGetPod sets up mock for successful pod retrieval
func (h *MockSetupHelper) SetupSuccessfulGetPod(mockSvc *MockPodService, namespace, name string) *services.PodInfo {
	pod := h.builder.CreatePodInfo(name, namespace)
	mockSvc.On("GetPod", mock.Anything, namespace, name).Return(&pod, nil)
	return &pod
}

// SetupPodNotFound sets up mock for pod not found scenario
func (h *MockSetupHelper) SetupPodNotFound(mockSvc *MockPodService, namespace, name string) {
	err := h.errors.NotFound("pods", name)
	mockSvc.On("GetPod", mock.Anything, namespace, name).Return((*services.PodInfo)(nil), err)
}

// SetupSuccessfulDeletePod sets up mock for successful pod deletion
func (h *MockSetupHelper) SetupSuccessfulDeletePod(mockSvc *MockPodService, namespace, name, user string) {
	mockSvc.On("DeletePod", mock.Anything, namespace, name, user).Return(nil)
}

// SetupDeletePodUnauthorized sets up mock for unauthorized pod deletion
func (h *MockSetupHelper) SetupDeletePodUnauthorized(mockSvc *MockPodService, namespace, name, user string) {
	err := h.errors.Unauthorized("insufficient permissions")
	mockSvc.On("DeletePod", mock.Anything, namespace, name, user).Return(err)
}

// SetupSuccessfulRestartPod sets up mock for successful pod restart
func (h *MockSetupHelper) SetupSuccessfulRestartPod(mockSvc *MockPodService, namespace, name, user, podType string) map[string]interface{} {
	result := map[string]interface{}{
		"message":   "Restart initiated",
		"type":      podType,
		"pod":       name,
		"namespace": namespace,
	}

	switch podType {
	case "deployment":
		result["deployment"] = fmt.Sprintf("%s-deployment", name)
	case "statefulset":
		result["statefulset"] = fmt.Sprintf("%s-statefulset", name)
	case "standalone":
		result["warning"] = "Pod will not be automatically recreated"
	}

	mockSvc.On("RestartPod", mock.Anything, namespace, name, user).Return(result, nil)
	return result
}

// SetupSuccessfulScaleDeployment sets up mock for successful deployment scaling
func (h *MockSetupHelper) SetupSuccessfulScaleDeployment(mockSvc *MockPodService, namespace, name, user string, oldReplicas, newReplicas int32) map[string]interface{} {
	result := map[string]interface{}{
		"message":          "Deployment scaled successfully",
		"deployment":       name,
		"previousReplicas": oldReplicas,
		"newReplicas":      newReplicas,
	}

	mockSvc.On("ScaleDeployment", mock.Anything, namespace, name, newReplicas, user).Return(result, nil)
	return result
}

// SetupSuccessfulWatchPods sets up mock for successful pod watching
func (h *MockSetupHelper) SetupSuccessfulWatchPods(mockSvc *MockPodService, namespace string) {
	mockSvc.On("WatchPods", mock.Anything, namespace, mock.AnythingOfType("chan<- services.PodWatchEvent")).Return(nil)
}

// ValidationTestCases provides common validation test cases
type ValidationTestCases struct{}

// NewValidationTestCases creates validation test cases helper
func NewValidationTestCases() *ValidationTestCases {
	return &ValidationTestCases{}
}

// InvalidNamespaces returns test cases for invalid namespace validation
func (v *ValidationTestCases) InvalidNamespaces() []struct {
	Name      string
	Namespace string
	Expected  string
} {
	return []struct {
		Name      string
		Namespace string
		Expected  string
	}{
		{"empty namespace", "", "namespace cannot be empty"},
		{"uppercase namespace", "INVALID", "invalid namespace name"},
		{"underscore namespace", "invalid_namespace", "invalid namespace name"},
		{"dot namespace", "invalid.namespace", "invalid namespace name"},
		{"reserved kube-system", "kube-system", "reserved namespace"},
		{"reserved kube-public", "kube-public", "reserved namespace"},
		{"reserved kube-node-lease", "kube-node-lease", "reserved namespace"},
		{"too long namespace", "this-namespace-name-is-way-too-long-and-exceeds-kubernetes-limits", "namespace name too long"},
		{"special characters", "test@namespace", "invalid namespace name"},
	}
}

// InvalidPodNames returns test cases for invalid pod name validation
func (v *ValidationTestCases) InvalidPodNames() []struct {
	Name     string
	PodName  string
	Expected string
} {
	return []struct {
		Name     string
		PodName  string
		Expected string
	}{
		{"empty pod name", "", "pod name cannot be empty"},
		{"uppercase pod name", "INVALID", "invalid pod name"},
		{"dangerous kube- prefix", "kube-proxy", "reserved prefix"},
		{"dangerous system- prefix", "system-admin", "reserved prefix"},
		{"too long pod name", "this-pod-name-is-way-too-long-and-exceeds-kubernetes-limits-test", "pod name too long"},
		{"underscore pod name", "invalid_pod", "invalid pod name"},
		{"special characters", "test@pod", "invalid pod name"},
	}
}

// InvalidUsers returns test cases for invalid user validation
func (v *ValidationTestCases) InvalidUsers() []struct {
	Name     string
	User     *string
	Expected string
} {
	return []struct {
		Name     string
		User     *string
		Expected string
	}{
		{"empty user", stringPtr(""), "empty string when provided"},
		{"script injection", stringPtr("user<script>"), "invalid characters"},
		{"quote injection", stringPtr("user\"admin"), "invalid characters"},
		{"ampersand", stringPtr("user&admin"), "invalid characters"},
		{"angle brackets", stringPtr("user>admin"), "invalid characters"},
		{"single quote", stringPtr("user'admin"), "invalid characters"},
		{"too long user", stringPtr(string(make([]byte, 101))), "user name too long"},
	}
}

// InvalidScaleInputs returns test cases for invalid scale input validation
func (v *ValidationTestCases) InvalidScaleInputs() []struct {
	Name     string
	Replicas int
	Expected string
} {
	return []struct {
		Name     string
		Replicas int
		Expected string
	}{
		{"negative replicas", -1, "cannot be negative"},
		{"too many replicas", 2000, "too high"},
		{"extremely high replicas", 10000, "too high"},
	}
}

// AuditTestHelper provides utilities for testing audit logging
type AuditTestHelper struct{}

// NewAuditTestHelper creates an audit test helper
func NewAuditTestHelper() *AuditTestHelper {
	return &AuditTestHelper{}
}

// ExpectMutationAudit sets up expected audit log calls for mutations
func (a *AuditTestHelper) ExpectMutationAudit(mockLogger *MockAuditLogger, operation, namespace, resource, user string, success bool, errorMsg string) {
	mockLogger.On("LogMutation", mock.Anything, operation, namespace, resource, user, success, errorMsg).Once()
}

// ExpectQueryAudit sets up expected audit log calls for queries
func (a *AuditTestHelper) ExpectQueryAudit(mockLogger *MockAuditLogger, operation, namespace string, resourceCount int) {
	mockLogger.On("LogQuery", mock.Anything, operation, namespace, resourceCount).Once()
}

// ExpectSubscriptionAudit sets up expected audit log calls for subscriptions
func (a *AuditTestHelper) ExpectSubscriptionAudit(mockLogger *MockAuditLogger, operation, namespace string, started bool) {
	mockLogger.On("LogSubscription", mock.Anything, operation, namespace, started).Once()
}

// ContextBuilder provides utilities for building test contexts
type ContextBuilder struct{}

// NewContextBuilder creates a new context builder
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// WithUser adds user to context
func (c *ContextBuilder) WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, "user", user)
}

// WithUserAgent adds user agent to context
func (c *ContextBuilder) WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, "user-agent", userAgent)
}

// WithRemoteAddr adds remote address to context
func (c *ContextBuilder) WithRemoteAddr(ctx context.Context, remoteAddr string) context.Context {
	return context.WithValue(ctx, "remote-addr", remoteAddr)
}

// WithTimeout creates context with timeout
func (c *ContextBuilder) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// PerformanceTestHelper provides utilities for performance testing
type PerformanceTestHelper struct{}

// NewPerformanceTestHelper creates a performance test helper
func NewPerformanceTestHelper() *PerformanceTestHelper {
	return &PerformanceTestHelper{}
}

// CreateLargeDataset creates large datasets for performance testing
func (p *PerformanceTestHelper) CreateLargeDataset(size int, namespace string) []services.PodInfo {
	builder := NewTestDataBuilder()
	pods := make([]services.PodInfo, size)

	for i := 0; i < size; i++ {
		pod := builder.CreatePodInfo(fmt.Sprintf("pod-%d", i), namespace)

		// Add variety to the dataset
		if i%10 == 0 {
			pod.Phase = corev1.PodPending
		} else if i%15 == 0 {
			pod.Phase = corev1.PodFailed
		}

		// Vary container count
		containerCount := (i % 3) + 1
		if containerCount > 1 {
			pod = builder.CreatePodInfoWithMultipleContainers(pod.Name, pod.Namespace, containerCount)
		}

		pods[i] = pod
	}

	return pods
}

// MeasureExecutionTime measures execution time of a function
func (p *PerformanceTestHelper) MeasureExecutionTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertExecutionTime asserts that execution time is within expected bounds
func (p *PerformanceTestHelper) AssertExecutionTime(duration, maxDuration time.Duration, operation string) bool {
	if duration > maxDuration {
		return false
	}
	return true
}

// TestResolverFactory provides factory methods for creating test resolvers
type TestResolverFactory struct{}

// NewTestResolverFactory creates a new test resolver factory
func NewTestResolverFactory() *TestResolverFactory {
	return &TestResolverFactory{}
}

// CreateMockedResolver creates a resolver with all mocked dependencies
func (f *TestResolverFactory) CreateMockedResolver() (*Resolver, *MockPodService, *MockHealthService, *MockAuditLogger) {
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
	return resolver, mockPodSvc, mockHealthSvc, mockAuditLogger
}

// CreateTestResolver creates a resolver with real implementations except for the pod service
func (f *TestResolverFactory) CreateTestResolver(mockPodSvc *MockPodService) *Resolver {
	return NewTestResolver(mockPodSvc)
}

// Helper functions for common operations
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

// Constants for testing
const (
	DefaultTestNamespace = "default"
	TestUser            = "test-user"
	SystemUser          = "system"
	AdminUser           = "admin"
	DeveloperUser       = "developer"
)